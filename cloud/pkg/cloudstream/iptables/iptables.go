package iptables

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilexec "k8s.io/utils/exec"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

type Manager struct {
	iptables            utiliptables.Interface
	cmLister            v1.ConfigMapLister
	cmListerSynced      cache.InformerSynced
	preTunnelPortRecord *TunnelPortRecord
	config              *v1alpha1.IptablesManager
}

type TunnelPortRecord struct {
	IPTunnelPort map[string]int `json:"ipTunnelPort"`
	Port         map[int]bool   `json:"port"`
}

type iptablesJumpChain struct {
	table     utiliptables.Table
	dstChain  utiliptables.Chain
	srcChain  utiliptables.Chain
	comment   string
	extraArgs []string
}

const (
	// the tunnelPort chain
	tunnelPortChain utiliptables.Chain = "TUNNEL-PORT"
)

var iptablesJumpChains = []iptablesJumpChain{
	{utiliptables.TableNAT, tunnelPortChain, utiliptables.ChainOutput, "kubeedge tunnel port", nil},
	{utiliptables.TableNAT, tunnelPortChain, utiliptables.ChainPrerouting, "kubeedge tunnel port", nil},
}

func NewIptablesManager(config *v1alpha1.IptablesManager) *Manager {
	protocol := utiliptables.ProtocolIPv4
	exec := utilexec.New()
	iptInterface := utiliptables.New(exec, protocol)

	iptablesMgr := &Manager{
		iptables: iptInterface,
		preTunnelPortRecord: &TunnelPortRecord{
			IPTunnelPort: make(map[string]int),
			Port:         make(map[int]bool),
		},
		config: config,
	}

	// informer factory
	k8sInformerFactory := informers.GetInformersManager().GetK8sInformerFactory()
	configMapsInformer := k8sInformerFactory.Core().V1().ConfigMaps()

	// lister
	iptablesMgr.cmLister = configMapsInformer.Lister()
	iptablesMgr.cmListerSynced = configMapsInformer.Informer().HasSynced

	return iptablesMgr
}

func (im *Manager) Run() {
	if !cache.WaitForCacheSync(beehiveContext.Done(), im.cmListerSynced) {
		klog.Error("unable to sync caches for iptables manager")
		return
	}

	err := im.iptables.FlushChain(utiliptables.TableNAT, tunnelPortChain)
	if err != nil {
		klog.Warningf("failed to delete all rules in tunnel port iptables chain: %v", err)
	}

	go wait.Until(im.reconcile, 10*time.Second, beehiveContext.Done())
}

func (im *Manager) reconcile() {
	// Create and link the tunnel port chains to OUTPUT and PREROUTING chain.
	for _, jump := range iptablesJumpChains {
		if _, err := im.iptables.EnsureChain(jump.table, jump.dstChain); err != nil {
			klog.ErrorS(err, "Failed to ensure chain exists", "table", jump.table, "chain", jump.dstChain)
			return
		}
		args := append(jump.extraArgs,
			"-m", "comment", "--comment", jump.comment,
			"-j", string(jump.dstChain),
		)
		if _, err := im.iptables.EnsureRule(utiliptables.Append, jump.table, jump.srcChain, args...); err != nil {
			klog.ErrorS(err, "Failed to ensure chain jumps", "table", jump.table, "srcChain", jump.srcChain, "dstChain", jump.dstChain)
			return
		}
	}

	addedIPPort, deletedIPPort, err := im.getAddedAndDeletedCloudCoreIPPort()
	if err != nil {
		klog.Errorf("failed to get added and deleted cloudcore ip and port in iptables manager: %v", err)
		return
	}

	var forwardPort uint32 = im.config.ForwardPort

	for _, ipports := range addedIPPort {
		ipport := strings.Split(ipports, ":")
		ip, port := ipport[0], ipport[1]
		args := append([]string{"-p", "tcp", "-j", "DNAT", "--dport", port, "--to", ip + ":" + strconv.Itoa(int(forwardPort))})
		if _, err := im.iptables.EnsureRule(utiliptables.Append, utiliptables.TableNAT, tunnelPortChain, args...); err != nil {
			klog.ErrorS(err, "Failed to ensure rules", "table", utiliptables.TableNAT, "chain", tunnelPortChain)
			return
		}
	}

	for _, ipports := range deletedIPPort {
		ipport := strings.Split(ipports, ":")
		ip, port := ipport[0], ipport[1]
		args := append([]string{"-p", "tcp", "-j", "DNAT", "--dport", port, "--to", ip + ":" + strconv.Itoa(int(forwardPort))})
		if err := im.iptables.DeleteRule(utiliptables.TableNAT, tunnelPortChain, args...); err != nil {
			klog.ErrorS(err, "Failed to delete rules", "table", utiliptables.TableNAT, "chain", tunnelPortChain)
			return
		}
	}
}

func (im *Manager) getAddedAndDeletedCloudCoreIPPort() ([]string, []string, error) {
	latestRecord, err := im.getLatestTunnelPortRecords()
	if err != nil {
		return nil, nil, err
	}

	addedIPPorts := []string{}
	for ip, port := range latestRecord.IPTunnelPort {
		if _, ok := im.preTunnelPortRecord.IPTunnelPort[ip]; !ok {
			addedIPPorts = append(addedIPPorts, strings.Join([]string{ip, strconv.Itoa(port)}, ":"))
		}
	}

	deletedIPPorts := []string{}
	for ip, port := range im.preTunnelPortRecord.IPTunnelPort {
		if _, ok := latestRecord.IPTunnelPort[ip]; !ok {
			deletedIPPorts = append(deletedIPPorts, strings.Join([]string{ip, strconv.Itoa(port)}, ":"))
		}
	}

	im.preTunnelPortRecord = latestRecord

	return addedIPPorts, deletedIPPorts, nil
}

func (im *Manager) getLatestTunnelPortRecords() (*TunnelPortRecord, error) {
	configmap, err := im.cmLister.ConfigMaps(constants.SystemNamespace).Get(modules.TunnelPort)
	if err != nil {
		return nil, errors.New("failed to get configmap for iptables manager")
	}

	recordStr, found := configmap.Annotations[modules.TunnelPortRecordAnnotationKey]
	recordBytes := []byte(recordStr)
	if !found {
		return nil, errors.New("failed to get tunnel port record")
	}

	record := &TunnelPortRecord{
		IPTunnelPort: make(map[string]int),
		Port:         make(map[int]bool),
	}
	if err := json.Unmarshal(recordBytes, record); err != nil {
		return nil, err
	}

	return record, nil
}
