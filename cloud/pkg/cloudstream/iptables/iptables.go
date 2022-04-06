package iptables

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
	cloudcoreConfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

type Manager struct {
	iptables              utiliptables.Interface
	sharedInformerFactory k8sinformer.SharedInformerFactory
	cmLister              v1.ConfigMapLister
	cmListerSynced        cache.InformerSynced
	preTunnelPortRecord   *TunnelPortRecord
	streamPort            int
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

var (
	iptablesJumpChains = []iptablesJumpChain{
		{utiliptables.TableNAT, tunnelPortChain, utiliptables.ChainOutput, "kubeedge tunnel port", nil},
		{utiliptables.TableNAT, tunnelPortChain, utiliptables.ChainPrerouting, "kubeedge tunnel port", nil},
	}
	kubeClient *kubernetes.Clientset
)

func NewIptablesManager(config *cloudcoreConfig.KubeAPIConfig, streamPort int) *Manager {
	protocol := utiliptables.ProtocolIPv4
	exec := utilexec.New()
	iptInterface := utiliptables.New(exec, protocol)

	iptablesMgr := &Manager{
		iptables: iptInterface,
		preTunnelPortRecord: &TunnelPortRecord{
			IPTunnelPort: make(map[string]int),
			Port:         make(map[int]bool),
		},
		streamPort: streamPort,
	}

	if kubeClient == nil {
		kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Master, config.KubeConfig)
		if err != nil {
			klog.Errorf("Failed to build config, err: %v", err)
			os.Exit(1)
		}
		kubeConfig.QPS = float32(config.QPS)
		kubeConfig.Burst = int(config.Burst)
		kubeConfig.ContentType = runtime.ContentTypeProtobuf
		kubeClient = kubernetes.NewForConfigOrDie(kubeConfig)
	}

	// informer factory
	k8sInformerFactory := k8sinformer.NewSharedInformerFactory(kubeClient, 0)
	configMapsInformer := k8sInformerFactory.Core().V1().ConfigMaps()

	// lister
	iptablesMgr.cmLister = configMapsInformer.Lister()
	iptablesMgr.cmListerSynced = configMapsInformer.Informer().HasSynced

	iptablesMgr.sharedInformerFactory = k8sInformerFactory
	return iptablesMgr
}

func (im *Manager) Run(ctx context.Context) {
	im.sharedInformerFactory.Start(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), im.cmListerSynced) {
		klog.Error("unable to sync caches for iptables manager")
		return
	}

	err := im.iptables.FlushChain(utiliptables.TableNAT, tunnelPortChain)
	if err != nil {
		klog.Warningf("failed to delete all rules in tunnel port iptables chain: %v", err)
	}

	go wait.Until(im.reconcile, 10*time.Second, ctx.Done())
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

	for _, ipports := range addedIPPort {
		ipport := strings.Split(ipports, ":")
		ip, port := ipport[0], ipport[1]
		args := append([]string{"-p", "tcp", "-j", "DNAT", "--dport", port, "--to", ip + ":" + strconv.Itoa(im.streamPort)})
		if _, err := im.iptables.EnsureRule(utiliptables.Append, utiliptables.TableNAT, tunnelPortChain, args...); err != nil {
			klog.ErrorS(err, "Failed to ensure rules", "table", utiliptables.TableNAT, "chain", tunnelPortChain)
			return
		}
	}

	for _, ipports := range deletedIPPort {
		ipport := strings.Split(ipports, ":")
		ip, port := ipport[0], ipport[1]
		args := append([]string{"-p", "tcp", "-j", "DNAT", "--dport", port, "--to", ip + ":" + strconv.Itoa(im.streamPort)})
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
		addedIPPorts = append(addedIPPorts, strings.Join([]string{ip, strconv.Itoa(port)}, ":"))
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
		return nil, errors.New("failed to get tunnelport configmap for iptables manager")
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
