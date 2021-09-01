// +build !dockerless

/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cni

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/containernetworking/cni/libcni"
	cnitypes "github.com/containernetworking/cni/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog/v2"
	kubeletconfig "k8s.io/kubernetes/pkg/kubelet/apis/config"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/dockershim/network"
	"k8s.io/kubernetes/pkg/util/bandwidth"
	utilslice "k8s.io/kubernetes/pkg/util/slice"
	utilexec "k8s.io/utils/exec"
)

const (
	// CNIPluginName is the name of CNI plugin
	CNIPluginName = "cni"

	// defaultSyncConfigPeriod is the default period to sync CNI config
	// TODO: consider making this value configurable or to be a more appropriate value.
	defaultSyncConfigPeriod = time.Second * 5

	// supported capabilities
	// https://github.com/containernetworking/cni/blob/master/CONVENTIONS.md
	portMappingsCapability = "portMappings"
	ipRangesCapability     = "ipRanges"
	bandwidthCapability    = "bandwidth"
	dnsCapability          = "dns"
)

type cniNetworkPlugin struct {
	network.NoopNetworkPlugin

	loNetwork *cniNetwork

	sync.RWMutex
	defaultNetwork *cniNetwork

	host        network.Host
	execer      utilexec.Interface
	nsenterPath string
	confDir     string
	binDirs     []string
	cacheDir    string
	podCidr     string
}

type cniNetwork struct {
	name          string
	NetworkConfig *libcni.NetworkConfigList
	CNIConfig     libcni.CNI
	Capabilities  []string
}

// cniPortMapping maps to the standard CNI portmapping Capability
// see: https://github.com/containernetworking/cni/blob/master/CONVENTIONS.md
type cniPortMapping struct {
	HostPort      int32  `json:"hostPort"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol"`
	HostIP        string `json:"hostIP"`
}

// cniBandwidthEntry maps to the standard CNI bandwidth Capability
// see: https://github.com/containernetworking/cni/blob/master/CONVENTIONS.md and
// https://github.com/containernetworking/plugins/blob/master/plugins/meta/bandwidth/README.md
type cniBandwidthEntry struct {
	// IngressRate is the bandwidth rate in bits per second for traffic through container. 0 for no limit. If IngressRate is set, IngressBurst must also be set
	IngressRate int `json:"ingressRate,omitempty"`
	// IngressBurst is the bandwidth burst in bits for traffic through container. 0 for no limit. If IngressBurst is set, IngressRate must also be set
	// NOTE: it's not used for now and defaults to 0. If IngressRate is set IngressBurst will be math.MaxInt32 ~ 2Gbit
	IngressBurst int `json:"ingressBurst,omitempty"`
	// EgressRate is the bandwidth is the bandwidth rate in bits per second for traffic through container. 0 for no limit. If EgressRate is set, EgressBurst must also be set
	EgressRate int `json:"egressRate,omitempty"`
	// EgressBurst is the bandwidth burst in bits for traffic through container. 0 for no limit. If EgressBurst is set, EgressRate must also be set
	// NOTE: it's not used for now and defaults to 0. If EgressRate is set EgressBurst will be math.MaxInt32 ~ 2Gbit
	EgressBurst int `json:"egressBurst,omitempty"`
}

// cniIPRange maps to the standard CNI ip range Capability
type cniIPRange struct {
	Subnet string `json:"subnet"`
}

// cniDNSConfig maps to the windows CNI dns Capability.
// see: https://github.com/containernetworking/cni/blob/master/CONVENTIONS.md
// Note that dns capability is only used for Windows containers.
type cniDNSConfig struct {
	// List of DNS servers of the cluster.
	Servers []string `json:"servers,omitempty"`
	// List of DNS search domains of the cluster.
	Searches []string `json:"searches,omitempty"`
	// List of DNS options.
	Options []string `json:"options,omitempty"`
}

// SplitDirs : split dirs by ","
func SplitDirs(dirs string) []string {
	// Use comma rather than colon to work better with Windows too
	return strings.Split(dirs, ",")
}

// ProbeNetworkPlugins : get the network plugin based on cni conf file and bin file
func ProbeNetworkPlugins(confDir, cacheDir string, binDirs []string) []network.NetworkPlugin {
	old := binDirs
	binDirs = make([]string, 0, len(binDirs))
	for _, dir := range old {
		if dir != "" {
			binDirs = append(binDirs, dir)
		}
	}

	plugin := &cniNetworkPlugin{
		defaultNetwork: nil,
		loNetwork:      getLoNetwork(binDirs),
		execer:         utilexec.New(),
		confDir:        confDir,
		binDirs:        binDirs,
		cacheDir:       cacheDir,
	}

	// sync NetworkConfig in best effort during probing.
	plugin.syncNetworkConfig()
	return []network.NetworkPlugin{plugin}
}

func getDefaultCNINetwork(confDir string, binDirs []string) (*cniNetwork, error) {
	files, err := libcni.ConfFiles(confDir, []string{".conf", ".conflist", ".json"})
	switch {
	case err != nil:
		return nil, err
	case len(files) == 0:
		return nil, fmt.Errorf("no networks found in %s", confDir)
	}

	cniConfig := &libcni.CNIConfig{Path: binDirs}

	sort.Strings(files)
	for _, confFile := range files {
		var confList *libcni.NetworkConfigList
		if strings.HasSuffix(confFile, ".conflist") {
			confList, err = libcni.ConfListFromFile(confFile)
			if err != nil {
				klog.InfoS("Error loading CNI config list file", "path", confFile, "err", err)
				continue
			}
		} else {
			conf, err := libcni.ConfFromFile(confFile)
			if err != nil {
				klog.InfoS("Error loading CNI config file", "path", confFile, "err", err)
				continue
			}
			// Ensure the config has a "type" so we know what plugin to run.
			// Also catches the case where somebody put a conflist into a conf file.
			if conf.Network.Type == "" {
				klog.InfoS("Error loading CNI config file: no 'type'; perhaps this is a .conflist?", "path", confFile)
				continue
			}

			confList, err = libcni.ConfListFromConf(conf)
			if err != nil {
				klog.InfoS("Error converting CNI config file to list", "path", confFile, "err", err)
				continue
			}
		}
		if len(confList.Plugins) == 0 {
			klog.InfoS("CNI config list has no networks, skipping", "configList", string(confList.Bytes[:maxStringLengthInLog(len(confList.Bytes))]))
			continue
		}

		// Before using this CNI config, we have to validate it to make sure that
		// all plugins of this config exist on disk
		caps, err := cniConfig.ValidateNetworkList(context.TODO(), confList)
		if err != nil {
			klog.InfoS("Error validating CNI config list", "configList", string(confList.Bytes[:maxStringLengthInLog(len(confList.Bytes))]), "err", err)
			continue
		}

		klog.V(4).InfoS("Using CNI configuration file", "path", confFile)

		return &cniNetwork{
			name:          confList.Name,
			NetworkConfig: confList,
			CNIConfig:     cniConfig,
			Capabilities:  caps,
		}, nil
	}
	return nil, fmt.Errorf("no valid networks found in %s", confDir)
}

func (plugin *cniNetworkPlugin) Init(host network.Host, hairpinMode kubeletconfig.HairpinMode, nonMasqueradeCIDR string, mtu int) error {
	err := plugin.platformInit()
	if err != nil {
		return err
	}

	plugin.host = host

	plugin.syncNetworkConfig()

	// start a goroutine to sync network config from confDir periodically to detect network config updates in every 5 seconds
	go wait.Forever(plugin.syncNetworkConfig, defaultSyncConfigPeriod)

	return nil
}

func (plugin *cniNetworkPlugin) syncNetworkConfig() {
	network, err := getDefaultCNINetwork(plugin.confDir, plugin.binDirs)
	if err != nil {
		klog.InfoS("Unable to update cni config", "err", err)
		return
	}
	plugin.setDefaultNetwork(network)
}

func (plugin *cniNetworkPlugin) getDefaultNetwork() *cniNetwork {
	plugin.RLock()
	defer plugin.RUnlock()
	return plugin.defaultNetwork
}

func (plugin *cniNetworkPlugin) setDefaultNetwork(n *cniNetwork) {
	plugin.Lock()
	defer plugin.Unlock()
	plugin.defaultNetwork = n
}

func (plugin *cniNetworkPlugin) checkInitialized() error {
	if plugin.getDefaultNetwork() == nil {
		return fmt.Errorf("cni config uninitialized")
	}

	if utilslice.ContainsString(plugin.getDefaultNetwork().Capabilities, ipRangesCapability, nil) && plugin.podCidr == "" {
		return fmt.Errorf("cni config needs ipRanges but no PodCIDR set")
	}

	return nil
}

// Event handles any change events. The only event ever sent is the PodCIDR change.
// No network plugins support changing an already-set PodCIDR
func (plugin *cniNetworkPlugin) Event(name string, details map[string]interface{}) {
	if name != network.NET_PLUGIN_EVENT_POD_CIDR_CHANGE {
		return
	}

	plugin.Lock()
	defer plugin.Unlock()

	podCIDR, ok := details[network.NET_PLUGIN_EVENT_POD_CIDR_CHANGE_DETAIL_CIDR].(string)
	if !ok {
		klog.InfoS("The event didn't contain pod CIDR", "event", network.NET_PLUGIN_EVENT_POD_CIDR_CHANGE)
		return
	}

	if plugin.podCidr != "" {
		klog.InfoS("Ignoring subsequent pod CIDR update to new cidr", "podCIDR", podCIDR)
		return
	}

	plugin.podCidr = podCIDR
}

func (plugin *cniNetworkPlugin) Name() string {
	return CNIPluginName
}

func (plugin *cniNetworkPlugin) Status() error {
	// Can't set up pods if we don't have any CNI network configs yet
	return plugin.checkInitialized()
}

func (plugin *cniNetworkPlugin) SetUpPod(namespace string, name string, id kubecontainer.ContainerID, annotations, options map[string]string) error {
	if err := plugin.checkInitialized(); err != nil {
		return err
	}
	netnsPath, err := plugin.host.GetNetNS(id.ID)
	if err != nil {
		return fmt.Errorf("CNI failed to retrieve network namespace path: %v", err)
	}

	// Todo get the timeout from parent ctx
	cniTimeoutCtx, cancelFunc := context.WithTimeout(context.Background(), network.CNITimeoutSec*time.Second)
	defer cancelFunc()
	// Windows doesn't have loNetwork. It comes only with Linux
	if plugin.loNetwork != nil {
		if _, err = plugin.addToNetwork(cniTimeoutCtx, plugin.loNetwork, name, namespace, id, netnsPath, annotations, options); err != nil {
			return err
		}
	}

	_, err = plugin.addToNetwork(cniTimeoutCtx, plugin.getDefaultNetwork(), name, namespace, id, netnsPath, annotations, options)
	return err
}

func (plugin *cniNetworkPlugin) TearDownPod(namespace string, name string, id kubecontainer.ContainerID) error {
	if err := plugin.checkInitialized(); err != nil {
		return err
	}

	// Lack of namespace should not be fatal on teardown
	netnsPath, err := plugin.host.GetNetNS(id.ID)
	if err != nil {
		klog.InfoS("CNI failed to retrieve network namespace path", "err", err)
	}

	// Todo get the timeout from parent ctx
	cniTimeoutCtx, cancelFunc := context.WithTimeout(context.Background(), network.CNITimeoutSec*time.Second)
	defer cancelFunc()
	// Windows doesn't have loNetwork. It comes only with Linux
	if plugin.loNetwork != nil {
		// Loopback network deletion failure should not be fatal on teardown
		if err := plugin.deleteFromNetwork(cniTimeoutCtx, plugin.loNetwork, name, namespace, id, netnsPath, nil); err != nil {
			klog.InfoS("CNI failed to delete loopback network", "err", err)
		}
	}

	return plugin.deleteFromNetwork(cniTimeoutCtx, plugin.getDefaultNetwork(), name, namespace, id, netnsPath, nil)
}

func (plugin *cniNetworkPlugin) addToNetwork(ctx context.Context, network *cniNetwork, podName string, podNamespace string, podSandboxID kubecontainer.ContainerID, podNetnsPath string, annotations, options map[string]string) (cnitypes.Result, error) {
	rt, err := plugin.buildCNIRuntimeConf(podName, podNamespace, podSandboxID, podNetnsPath, annotations, options)
	if err != nil {
		klog.ErrorS(err, "Error adding network when building cni runtime conf")
		return nil, err
	}

	netConf, cniNet := network.NetworkConfig, network.CNIConfig
	klog.V(4).InfoS("Adding pod to network", "pod", klog.KRef(podNamespace, podName), "podSandboxID", podSandboxID, "podNetnsPath", podNetnsPath, "networkType", netConf.Plugins[0].Network.Type, "networkName", netConf.Name)
	res, err := cniNet.AddNetworkList(ctx, netConf, rt)
	if err != nil {
		klog.ErrorS(err, "Error adding pod to network", "pod", klog.KRef(podNamespace, podName), "podSandboxID", podSandboxID, "podNetnsPath", podNetnsPath, "networkType", netConf.Plugins[0].Network.Type, "networkName", netConf.Name)
		return nil, err
	}
	klog.V(4).InfoS("Added pod to network", "pod", klog.KRef(podNamespace, podName), "podSandboxID", podSandboxID, "networkName", netConf.Name, "response", res)
	return res, nil
}

func (plugin *cniNetworkPlugin) deleteFromNetwork(ctx context.Context, network *cniNetwork, podName string, podNamespace string, podSandboxID kubecontainer.ContainerID, podNetnsPath string, annotations map[string]string) error {
	rt, err := plugin.buildCNIRuntimeConf(podName, podNamespace, podSandboxID, podNetnsPath, annotations, nil)
	if err != nil {
		klog.ErrorS(err, "Error deleting network when building cni runtime conf")
		return err
	}
	netConf, cniNet := network.NetworkConfig, network.CNIConfig
	klog.V(4).InfoS("Deleting pod from network", "pod", klog.KRef(podNamespace, podName), "podSandboxID", podSandboxID, "podNetnsPath", podNetnsPath, "networkType", netConf.Plugins[0].Network.Type, "networkName", netConf.Name)
	err = cniNet.DelNetworkList(ctx, netConf, rt)
	// The pod may not get deleted successfully at the first time.
	// Ignore "no such file or directory" error in case the network has already been deleted in previous attempts.
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		klog.ErrorS(err, "Error deleting pod from network", "pod", klog.KRef(podNamespace, podName), "podSandboxID", podSandboxID, "podNetnsPath", podNetnsPath, "networkType", netConf.Plugins[0].Network.Type, "networkName", netConf.Name)
		return err
	}
	klog.V(4).InfoS("Deleted pod from network", "pod", klog.KRef(podNamespace, podName), "podSandboxID", podSandboxID, "networkType", netConf.Plugins[0].Network.Type, "networkName", netConf.Name)
	return nil
}

func (plugin *cniNetworkPlugin) buildCNIRuntimeConf(podName string, podNs string, podSandboxID kubecontainer.ContainerID, podNetnsPath string, annotations, options map[string]string) (*libcni.RuntimeConf, error) {
	rt := &libcni.RuntimeConf{
		ContainerID: podSandboxID.ID,
		NetNS:       podNetnsPath,
		IfName:      network.DefaultInterfaceName,
		CacheDir:    plugin.cacheDir,
		Args: [][2]string{
			{"IgnoreUnknown", "1"},
			{"K8S_POD_NAMESPACE", podNs},
			{"K8S_POD_NAME", podName},
			{"K8S_POD_INFRA_CONTAINER_ID", podSandboxID.ID},
		},
	}

	// port mappings are a cni capability-based args, rather than parameters
	// to a specific plugin
	portMappings, err := plugin.host.GetPodPortMappings(podSandboxID.ID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve port mappings: %v", err)
	}
	portMappingsParam := make([]cniPortMapping, 0, len(portMappings))
	for _, p := range portMappings {
		if p.HostPort <= 0 {
			continue
		}
		portMappingsParam = append(portMappingsParam, cniPortMapping{
			HostPort:      p.HostPort,
			ContainerPort: p.ContainerPort,
			Protocol:      strings.ToLower(string(p.Protocol)),
			HostIP:        p.HostIP,
		})
	}
	rt.CapabilityArgs = map[string]interface{}{
		portMappingsCapability: portMappingsParam,
	}

	ingress, egress, err := bandwidth.ExtractPodBandwidthResources(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod bandwidth from annotations: %v", err)
	}
	if ingress != nil || egress != nil {
		bandwidthParam := cniBandwidthEntry{}
		if ingress != nil {
			// see: https://github.com/containernetworking/cni/blob/master/CONVENTIONS.md and
			// https://github.com/containernetworking/plugins/blob/master/plugins/meta/bandwidth/README.md
			// Rates are in bits per second, burst values are in bits.
			bandwidthParam.IngressRate = int(ingress.Value())
			// Limit IngressBurst to math.MaxInt32, in practice limiting to 2Gbit is the equivalent of setting no limit
			bandwidthParam.IngressBurst = math.MaxInt32
		}
		if egress != nil {
			bandwidthParam.EgressRate = int(egress.Value())
			// Limit EgressBurst to math.MaxInt32, in practice limiting to 2Gbit is the equivalent of setting no limit
			bandwidthParam.EgressBurst = math.MaxInt32
		}
		rt.CapabilityArgs[bandwidthCapability] = bandwidthParam
	}

	// Set the PodCIDR
	rt.CapabilityArgs[ipRangesCapability] = [][]cniIPRange{{{Subnet: plugin.podCidr}}}

	// Set dns capability args.
	if dnsOptions, ok := options["dns"]; ok {
		dnsConfig := runtimeapi.DNSConfig{}
		err := json.Unmarshal([]byte(dnsOptions), &dnsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal dns config %q: %v", dnsOptions, err)
		}
		if dnsParam := buildDNSCapabilities(&dnsConfig); dnsParam != nil {
			rt.CapabilityArgs[dnsCapability] = *dnsParam
		}
	}

	return rt, nil
}

func maxStringLengthInLog(length int) int {
	// we allow no more than 4096-length strings to be logged
	const maxStringLength = 4096

	if length < maxStringLength {
		return length
	}
	return maxStringLength
}
