// +build linux,!dockerless

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

package kubenet

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/containernetworking/cni/libcni"
	cnitypes "github.com/containernetworking/cni/pkg/types"
	cnitypes020 "github.com/containernetworking/cni/pkg/types/020"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	utilsets "k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	kubeletconfig "k8s.io/kubernetes/pkg/kubelet/apis/config"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/dockershim/network"
	"k8s.io/kubernetes/pkg/kubelet/dockershim/network/hostport"
	"k8s.io/kubernetes/pkg/util/bandwidth"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilsysctl "k8s.io/kubernetes/pkg/util/sysctl"
	utilexec "k8s.io/utils/exec"
	utilebtables "k8s.io/utils/net/ebtables"

	utilfeature "k8s.io/apiserver/pkg/util/feature"
	kubefeatures "k8s.io/kubernetes/pkg/features"
	netutils "k8s.io/utils/net"
)

const (
	BridgeName    = "cbr0"
	DefaultCNIDir = "/opt/cni/bin"

	sysctlBridgeCallIPTables = "net/bridge/bridge-nf-call-iptables"

	// fallbackMTU is used if an MTU is not specified, and we cannot determine the MTU
	fallbackMTU = 1460

	// ebtables Chain to store dedup rules
	dedupChain = utilebtables.Chain("KUBE-DEDUP")

	zeroCIDRv6 = "::/0"
	zeroCIDRv4 = "0.0.0.0/0"

	NET_CONFIG_TEMPLATE = `{
  "cniVersion": "0.1.0",
  "name": "kubenet",
  "type": "bridge",
  "bridge": "%s",
  "mtu": %d,
  "addIf": "%s",
  "isGateway": true,
  "ipMasq": false,
  "hairpinMode": %t,
  "ipam": {
    "type": "host-local",
    "ranges": [%s],
    "routes": [%s]
  }
}`
)

// CNI plugins required by kubenet in /opt/cni/bin or user-specified directory
var requiredCNIPlugins = [...]string{"bridge", "host-local", "loopback"}

type kubenetNetworkPlugin struct {
	network.NoopNetworkPlugin

	host              network.Host
	netConfig         *libcni.NetworkConfig
	loConfig          *libcni.NetworkConfig
	cniConfig         libcni.CNI
	bandwidthShaper   bandwidth.Shaper
	mu                sync.Mutex //Mutex for protecting podIPs map, netConfig, and shaper initialization
	podIPs            map[kubecontainer.ContainerID]utilsets.String
	mtu               int
	execer            utilexec.Interface
	nsenterPath       string
	hairpinMode       kubeletconfig.HairpinMode
	hostportManager   hostport.HostPortManager
	hostportManagerv6 hostport.HostPortManager
	iptables          utiliptables.Interface
	iptablesv6        utiliptables.Interface
	sysctl            utilsysctl.Interface
	ebtables          utilebtables.Interface
	// binDirs is passed by kubelet cni-bin-dir parameter.
	// kubenet will search for CNI binaries in DefaultCNIDir first, then continue to binDirs.
	binDirs           []string
	nonMasqueradeCIDR string
	cacheDir          string
	podCIDRs          []*net.IPNet
}

func NewPlugin(networkPluginDirs []string, cacheDir string) network.NetworkPlugin {
	execer := utilexec.New()
	iptInterface := utiliptables.New(execer, utiliptables.ProtocolIPv4)
	iptInterfacev6 := utiliptables.New(execer, utiliptables.ProtocolIPv6)
	return &kubenetNetworkPlugin{
		podIPs:            make(map[kubecontainer.ContainerID]utilsets.String),
		execer:            utilexec.New(),
		iptables:          iptInterface,
		iptablesv6:        iptInterfacev6,
		sysctl:            utilsysctl.New(),
		binDirs:           append([]string{DefaultCNIDir}, networkPluginDirs...),
		hostportManager:   hostport.NewHostportManager(iptInterface),
		hostportManagerv6: hostport.NewHostportManager(iptInterfacev6),
		nonMasqueradeCIDR: "10.0.0.0/8",
		cacheDir:          cacheDir,
		podCIDRs:          make([]*net.IPNet, 0),
	}
}

func (plugin *kubenetNetworkPlugin) Init(host network.Host, hairpinMode kubeletconfig.HairpinMode, nonMasqueradeCIDR string, mtu int) error {
	plugin.host = host
	plugin.hairpinMode = hairpinMode
	plugin.nonMasqueradeCIDR = nonMasqueradeCIDR
	plugin.cniConfig = &libcni.CNIConfig{Path: plugin.binDirs}

	if mtu == network.UseDefaultMTU {
		if link, err := findMinMTU(); err == nil {
			plugin.mtu = link.MTU
			klog.V(5).InfoS("Using the interface MTU value as bridge MTU", "interfaceName", link.Name, "mtuValue", link.MTU)
		} else {
			plugin.mtu = fallbackMTU
			klog.InfoS("Failed to find default bridge MTU, using default value", "mtuValue", fallbackMTU, "err", err)
		}
	} else {
		plugin.mtu = mtu
	}

	// Since this plugin uses a Linux bridge, set bridge-nf-call-iptables=1
	// is necessary to ensure kube-proxy functions correctly.
	//
	// This will return an error on older kernel version (< 3.18) as the module
	// was built-in, we simply ignore the error here. A better thing to do is
	// to check the kernel version in the future.
	plugin.execer.Command("modprobe", "br-netfilter").CombinedOutput()
	err := plugin.sysctl.SetSysctl(sysctlBridgeCallIPTables, 1)
	if err != nil {
		klog.InfoS("can't set sysctl bridge-nf-call-iptables", "err", err)
	}

	plugin.loConfig, err = libcni.ConfFromBytes([]byte(`{
  "cniVersion": "0.1.0",
  "name": "kubenet-loopback",
  "type": "loopback"
}`))
	if err != nil {
		return fmt.Errorf("failed to generate loopback config: %v", err)
	}

	plugin.nsenterPath, err = plugin.execer.LookPath("nsenter")
	if err != nil {
		return fmt.Errorf("failed to find nsenter binary: %v", err)
	}

	// Need to SNAT outbound traffic from cluster
	if err = plugin.ensureMasqRule(); err != nil {
		return err
	}
	return nil
}

// TODO: move thic logic into cni bridge plugin and remove this from kubenet
func (plugin *kubenetNetworkPlugin) ensureMasqRule() error {
	if plugin.nonMasqueradeCIDR != zeroCIDRv4 && plugin.nonMasqueradeCIDR != zeroCIDRv6 {
		// switch according to target nonMasqueradeCidr ip family
		ipt := plugin.iptables
		if netutils.IsIPv6CIDRString(plugin.nonMasqueradeCIDR) {
			ipt = plugin.iptablesv6
		}

		if _, err := ipt.EnsureRule(utiliptables.Append, utiliptables.TableNAT, utiliptables.ChainPostrouting,
			"-m", "comment", "--comment", "kubenet: SNAT for outbound traffic from cluster",
			"-m", "addrtype", "!", "--dst-type", "LOCAL",
			"!", "-d", plugin.nonMasqueradeCIDR,
			"-j", "MASQUERADE"); err != nil {
			return fmt.Errorf("failed to ensure that %s chain %s jumps to MASQUERADE: %v", utiliptables.TableNAT, utiliptables.ChainPostrouting, err)
		}
	}
	return nil
}

func findMinMTU() (*net.Interface, error) {
	intfs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	mtu := 999999
	defIntfIndex := -1
	for i, intf := range intfs {
		if ((intf.Flags & net.FlagUp) != 0) && (intf.Flags&(net.FlagLoopback|net.FlagPointToPoint) == 0) {
			if intf.MTU < mtu {
				mtu = intf.MTU
				defIntfIndex = i
			}
		}
	}

	if mtu >= 999999 || mtu < 576 || defIntfIndex < 0 {
		return nil, fmt.Errorf("no suitable interface: %v", BridgeName)
	}

	return &intfs[defIntfIndex], nil
}

func (plugin *kubenetNetworkPlugin) Event(name string, details map[string]interface{}) {
	var err error
	if name != network.NET_PLUGIN_EVENT_POD_CIDR_CHANGE {
		return
	}

	plugin.mu.Lock()
	defer plugin.mu.Unlock()

	podCIDR, ok := details[network.NET_PLUGIN_EVENT_POD_CIDR_CHANGE_DETAIL_CIDR].(string)
	if !ok {
		klog.InfoS("The event didn't contain pod CIDR", "event", network.NET_PLUGIN_EVENT_POD_CIDR_CHANGE)
		return
	}

	if plugin.netConfig != nil {
		klog.InfoS("Ignoring subsequent pod CIDR update to new cidr", "podCIDR", podCIDR)
		return
	}

	klog.V(4).InfoS("Kubenet: PodCIDR is set to new value", "podCIDR", podCIDR)
	podCIDRs := strings.Split(podCIDR, ",")

	// reset to one cidr if dual stack is not enabled
	if !utilfeature.DefaultFeatureGate.Enabled(kubefeatures.IPv6DualStack) && len(podCIDRs) > 1 {
		klog.V(2).InfoS("This node has multiple pod cidrs assigned and dual stack is not enabled. ignoring all except first cidr")
		podCIDRs = podCIDRs[0:1]
	}

	for idx, currentPodCIDR := range podCIDRs {
		_, cidr, err := net.ParseCIDR(currentPodCIDR)
		if nil != err {
			klog.InfoS("Failed to generate CNI network config with cidr at the index", "podCIDR", currentPodCIDR, "index", idx, "err", err)
			return
		}
		// create list of ips
		plugin.podCIDRs = append(plugin.podCIDRs, cidr)
	}

	//setup hairpinMode
	setHairpin := plugin.hairpinMode == kubeletconfig.HairpinVeth

	json := fmt.Sprintf(NET_CONFIG_TEMPLATE, BridgeName, plugin.mtu, network.DefaultInterfaceName, setHairpin, plugin.getRangesConfig(), plugin.getRoutesConfig())
	klog.V(4).InfoS("CNI network config set to json format", "cniNetworkConfig", json)
	plugin.netConfig, err = libcni.ConfFromBytes([]byte(json))
	if err != nil {
		klog.InfoS("** failed to set up CNI with json format", "cniNetworkConfig", json, "err", err)
		// just incase it was set by mistake
		plugin.netConfig = nil
		// we bail out by clearing the *entire* list
		// of addresses assigned to cbr0
		plugin.clearUnusedBridgeAddresses()
	}
}

// clear all address on bridge except those operated on by kubenet
func (plugin *kubenetNetworkPlugin) clearUnusedBridgeAddresses() {
	cidrIncluded := func(list []*net.IPNet, check *net.IPNet) bool {
		for _, thisNet := range list {
			if utilnet.IPNetEqual(thisNet, check) {
				return true
			}
		}
		return false
	}

	bridge, err := netlink.LinkByName(BridgeName)
	if err != nil {
		return
	}

	addrs, err := netlink.AddrList(bridge, unix.AF_INET)
	if err != nil {
		klog.V(2).InfoS("Attempting to get address for the interface failed", "interfaceName", BridgeName, "err", err)
		return
	}

	for _, addr := range addrs {
		if !cidrIncluded(plugin.podCIDRs, addr.IPNet) {
			klog.V(2).InfoS("Removing old address from the interface", "interfaceName", BridgeName, "address", addr.IPNet.String())
			netlink.AddrDel(bridge, &addr)
		}
	}
}

func (plugin *kubenetNetworkPlugin) Name() string {
	return KubenetPluginName
}

func (plugin *kubenetNetworkPlugin) Capabilities() utilsets.Int {
	return utilsets.NewInt()
}

// setup sets up networking through CNI using the given ns/name and sandbox ID.
func (plugin *kubenetNetworkPlugin) setup(namespace string, name string, id kubecontainer.ContainerID, annotations map[string]string) error {
	var ipv4, ipv6 net.IP
	var podGateways []net.IP
	var podCIDRs []net.IPNet

	// Disable DAD so we skip the kernel delay on bringing up new interfaces.
	if err := plugin.disableContainerDAD(id); err != nil {
		klog.V(3).InfoS("Failed to disable DAD in container", "err", err)
	}

	// Bring up container loopback interface
	if _, err := plugin.addContainerToNetwork(plugin.loConfig, "lo", namespace, name, id); err != nil {
		return err
	}

	// Hook container up with our bridge
	resT, err := plugin.addContainerToNetwork(plugin.netConfig, network.DefaultInterfaceName, namespace, name, id)
	if err != nil {
		return err
	}
	// Coerce the CNI result version
	res, err := cnitypes020.GetResult(resT)
	if err != nil {
		return fmt.Errorf("unable to understand network config: %v", err)
	}
	//TODO: v1.16 (khenidak) update NET_CONFIG_TEMPLATE to CNI version 0.3.0 or later so
	// that we get multiple IP addresses in the returned Result structure
	if res.IP4 != nil {
		ipv4 = res.IP4.IP.IP.To4()
		podGateways = append(podGateways, res.IP4.Gateway)
		podCIDRs = append(podCIDRs, net.IPNet{IP: ipv4.Mask(res.IP4.IP.Mask), Mask: res.IP4.IP.Mask})
	}

	if res.IP6 != nil {
		ipv6 = res.IP6.IP.IP
		podGateways = append(podGateways, res.IP6.Gateway)
		podCIDRs = append(podCIDRs, net.IPNet{IP: ipv6.Mask(res.IP6.IP.Mask), Mask: res.IP6.IP.Mask})
	}

	if ipv4 == nil && ipv6 == nil {
		return fmt.Errorf("cni didn't report ipv4 ipv6")
	}
	// Put the container bridge into promiscuous mode to force it to accept hairpin packets.
	// TODO: Remove this once the kernel bug (#20096) is fixed.
	if plugin.hairpinMode == kubeletconfig.PromiscuousBridge {
		link, err := netlink.LinkByName(BridgeName)
		if err != nil {
			return fmt.Errorf("failed to lookup %q: %v", BridgeName, err)
		}
		if link.Attrs().Promisc != 1 {
			// promiscuous mode is not on, then turn it on.
			err := netlink.SetPromiscOn(link)
			if err != nil {
				return fmt.Errorf("error setting promiscuous mode on %s: %v", BridgeName, err)
			}
		}

		// configure the ebtables rules to eliminate duplicate packets by best effort
		plugin.syncEbtablesDedupRules(link.Attrs().HardwareAddr, podCIDRs, podGateways)
	}

	// add the ip to tracked ips
	if ipv4 != nil {
		plugin.addPodIP(id, ipv4.String())
	}
	if ipv6 != nil {
		plugin.addPodIP(id, ipv6.String())
	}

	if err := plugin.addTrafficShaping(id, annotations); err != nil {
		return err
	}

	return plugin.addPortMapping(id, name, namespace)
}

// The first SetUpPod call creates the bridge; get a shaper for the sake of initialization
// TODO: replace with CNI traffic shaper plugin
func (plugin *kubenetNetworkPlugin) addTrafficShaping(id kubecontainer.ContainerID, annotations map[string]string) error {
	shaper := plugin.shaper()
	ingress, egress, err := bandwidth.ExtractPodBandwidthResources(annotations)
	if err != nil {
		return fmt.Errorf("error reading pod bandwidth annotations: %v", err)
	}
	iplist, exists := plugin.getCachedPodIPs(id)
	if !exists {
		return fmt.Errorf("pod %s does not have recorded ips", id)
	}

	if egress != nil || ingress != nil {
		for _, ip := range iplist {
			mask := 32
			if netutils.IsIPv6String(ip) {
				mask = 128
			}
			if err != nil {
				return fmt.Errorf("failed to setup traffic shaping for pod ip%s", ip)
			}

			if err := shaper.ReconcileCIDR(fmt.Sprintf("%v/%v", ip, mask), egress, ingress); err != nil {
				return fmt.Errorf("failed to add pod to shaper: %v", err)
			}
		}
	}
	return nil
}

// TODO: replace with CNI port-forwarding plugin
func (plugin *kubenetNetworkPlugin) addPortMapping(id kubecontainer.ContainerID, name, namespace string) error {
	portMappings, err := plugin.host.GetPodPortMappings(id.ID)
	if err != nil {
		return err
	}

	if len(portMappings) == 0 {
		return nil
	}

	iplist, exists := plugin.getCachedPodIPs(id)
	if !exists {
		return fmt.Errorf("pod %s does not have recorded ips", id)
	}

	for _, ip := range iplist {
		pm := &hostport.PodPortMapping{
			Namespace:    namespace,
			Name:         name,
			PortMappings: portMappings,
			IP:           net.ParseIP(ip),
			HostNetwork:  false,
		}
		if netutils.IsIPv6(pm.IP) {
			if err := plugin.hostportManagerv6.Add(id.ID, pm, BridgeName); err != nil {
				return err
			}
		} else {
			if err := plugin.hostportManager.Add(id.ID, pm, BridgeName); err != nil {
				return err
			}
		}
	}

	return nil
}

func (plugin *kubenetNetworkPlugin) SetUpPod(namespace string, name string, id kubecontainer.ContainerID, annotations, options map[string]string) error {
	start := time.Now()

	if err := plugin.Status(); err != nil {
		return fmt.Errorf("kubenet cannot SetUpPod: %v", err)
	}

	defer func() {
		klog.V(4).InfoS("SetUpPod took time", "pod", klog.KRef(namespace, name), "duration", time.Since(start))
	}()

	if err := plugin.setup(namespace, name, id, annotations); err != nil {
		if err := plugin.teardown(namespace, name, id); err != nil {
			// Not a hard error or warning
			klog.V(4).InfoS("Failed to clean up pod after SetUpPod failure", "pod", klog.KRef(namespace, name), "err", err)
		}
		return err
	}

	// Need to SNAT outbound traffic from cluster
	if err := plugin.ensureMasqRule(); err != nil {
		klog.ErrorS(err, "Failed to ensure MASQ rule")
	}

	return nil
}

// Tears down as much of a pod's network as it can even if errors occur.  Returns
// an aggregate error composed of all errors encountered during the teardown.
func (plugin *kubenetNetworkPlugin) teardown(namespace string, name string, id kubecontainer.ContainerID) error {
	errList := []error{}

	// Loopback network deletion failure should not be fatal on teardown
	if err := plugin.delContainerFromNetwork(plugin.loConfig, "lo", namespace, name, id); err != nil {
		klog.InfoS("Failed to delete loopback network", "err", err)
		errList = append(errList, err)

	}

	// no ip dependent actions
	if err := plugin.delContainerFromNetwork(plugin.netConfig, network.DefaultInterfaceName, namespace, name, id); err != nil {
		klog.InfoS("Failed to delete the interface network", "interfaceName", network.DefaultInterfaceName, "err", err)
		errList = append(errList, err)
	}

	// If there are no IPs registered we can't teardown pod's IP dependencies
	iplist, exists := plugin.getCachedPodIPs(id)
	if !exists || len(iplist) == 0 {
		klog.V(5).InfoS("Container does not have IP registered. Ignoring teardown call", "containerID", id, "pod", klog.KRef(namespace, name))
		return nil
	}

	// get the list of port mappings
	portMappings, err := plugin.host.GetPodPortMappings(id.ID)
	if err != nil {
		errList = append(errList, err)
	}

	// process each pod IP
	for _, ip := range iplist {
		isV6 := netutils.IsIPv6String(ip)
		klog.V(5).InfoS("Removing pod port mappings from the IP", "IP", ip)
		if portMappings != nil && len(portMappings) > 0 {
			if isV6 {
				if err = plugin.hostportManagerv6.Remove(id.ID, &hostport.PodPortMapping{
					Namespace:    namespace,
					Name:         name,
					PortMappings: portMappings,
					HostNetwork:  false,
				}); err != nil {
					errList = append(errList, err)
				}
			} else {
				if err = plugin.hostportManager.Remove(id.ID, &hostport.PodPortMapping{
					Namespace:    namespace,
					Name:         name,
					PortMappings: portMappings,
					HostNetwork:  false,
				}); err != nil {
					errList = append(errList, err)
				}
			}
		}

		klog.V(5).InfoS("Removing pod IP from shaper for the pod", "pod", klog.KRef(namespace, name), "IP", ip)
		// shaper uses a cidr, but we are using a single IP.
		mask := "32"
		if isV6 {
			mask = "128"
		}

		if err := plugin.shaper().Reset(fmt.Sprintf("%s/%s", ip, mask)); err != nil {
			// Possible bandwidth shaping wasn't enabled for this pod anyways
			klog.V(4).InfoS("Failed to remove pod IP from shaper", "IP", ip, "err", err)
		}

		plugin.removePodIP(id, ip)
	}
	return utilerrors.NewAggregate(errList)
}

func (plugin *kubenetNetworkPlugin) TearDownPod(namespace string, name string, id kubecontainer.ContainerID) error {
	start := time.Now()
	defer func() {
		klog.V(4).InfoS("TearDownPod took time", "pod", klog.KRef(namespace, name), "duration", time.Since(start))
	}()

	if plugin.netConfig == nil {
		return fmt.Errorf("kubenet needs a PodCIDR to tear down pods")
	}

	if err := plugin.teardown(namespace, name, id); err != nil {
		return err
	}

	// Need to SNAT outbound traffic from cluster
	if err := plugin.ensureMasqRule(); err != nil {
		klog.ErrorS(err, "Failed to ensure MASQ rule")
	}
	return nil
}

// TODO: Use the addToNetwork function to obtain the IP of the Pod. That will assume idempotent ADD call to the plugin.
// Also fix the runtime's call to Status function to be done only in the case that the IP is lost, no need to do periodic calls
func (plugin *kubenetNetworkPlugin) GetPodNetworkStatus(namespace string, name string, id kubecontainer.ContainerID) (*network.PodNetworkStatus, error) {
	// try cached version
	networkStatus := plugin.getNetworkStatus(id)
	if networkStatus != nil {
		return networkStatus, nil
	}

	// not a cached version, get via network ns
	netnsPath, err := plugin.host.GetNetNS(id.ID)
	if err != nil {
		return nil, fmt.Errorf("kubenet failed to retrieve network namespace path: %v", err)
	}
	if netnsPath == "" {
		return nil, fmt.Errorf("cannot find the network namespace, skipping pod network status for container %q", id)
	}
	ips, err := network.GetPodIPs(plugin.execer, plugin.nsenterPath, netnsPath, network.DefaultInterfaceName)
	if err != nil {
		return nil, err
	}

	// cache the ips
	for _, ip := range ips {
		plugin.addPodIP(id, ip.String())
	}

	// return from cached
	return plugin.getNetworkStatus(id), nil
}

// returns networkstatus
func (plugin *kubenetNetworkPlugin) getNetworkStatus(id kubecontainer.ContainerID) *network.PodNetworkStatus {
	// Assuming the ip of pod does not change. Try to retrieve ip from kubenet map first.
	iplist, ok := plugin.getCachedPodIPs(id)
	if !ok {
		return nil
	}

	if len(iplist) == 0 {
		return nil
	}

	ips := make([]net.IP, 0, len(iplist))
	for _, ip := range iplist {
		ips = append(ips, net.ParseIP(ip))
	}

	return &network.PodNetworkStatus{
		IP:  ips[0],
		IPs: ips,
	}
}

func (plugin *kubenetNetworkPlugin) Status() error {
	// Can't set up pods if we don't have a PodCIDR yet
	if plugin.netConfig == nil {
		return fmt.Errorf("kubenet does not have netConfig. This is most likely due to lack of PodCIDR")
	}

	if !plugin.checkRequiredCNIPlugins() {
		return fmt.Errorf("could not locate kubenet required CNI plugins %v at %q", requiredCNIPlugins, plugin.binDirs)
	}
	return nil
}

// checkRequiredCNIPlugins returns if all kubenet required cni plugins can be found at /opt/cni/bin or user specified NetworkPluginDir.
func (plugin *kubenetNetworkPlugin) checkRequiredCNIPlugins() bool {
	for _, dir := range plugin.binDirs {
		if plugin.checkRequiredCNIPluginsInOneDir(dir) {
			return true
		}
	}
	return false
}

// checkRequiredCNIPluginsInOneDir returns true if all required cni plugins are placed in dir
func (plugin *kubenetNetworkPlugin) checkRequiredCNIPluginsInOneDir(dir string) bool {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, cniPlugin := range requiredCNIPlugins {
		found := false
		for _, file := range files {
			if strings.TrimSpace(file.Name()) == cniPlugin {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (plugin *kubenetNetworkPlugin) buildCNIRuntimeConf(ifName string, id kubecontainer.ContainerID, needNetNs bool) (*libcni.RuntimeConf, error) {
	netnsPath, err := plugin.host.GetNetNS(id.ID)
	if needNetNs && err != nil {
		klog.ErrorS(err, "Kubenet failed to retrieve network namespace path")
	}

	return &libcni.RuntimeConf{
		ContainerID: id.ID,
		NetNS:       netnsPath,
		IfName:      ifName,
		CacheDir:    plugin.cacheDir,
	}, nil
}

func (plugin *kubenetNetworkPlugin) addContainerToNetwork(config *libcni.NetworkConfig, ifName, namespace, name string, id kubecontainer.ContainerID) (cnitypes.Result, error) {
	rt, err := plugin.buildCNIRuntimeConf(ifName, id, true)
	if err != nil {
		return nil, fmt.Errorf("error building CNI config: %v", err)
	}

	klog.V(3).InfoS("Adding pod to network with CNI plugin and runtime", "pod", klog.KRef(namespace, name), "networkName", config.Network.Name, "networkType", config.Network.Type, "rt", rt)
	// Because the default remote runtime request timeout is 4 min,so set slightly less than 240 seconds
	// Todo get the timeout from parent ctx
	cniTimeoutCtx, cancelFunc := context.WithTimeout(context.Background(), network.CNITimeoutSec*time.Second)
	defer cancelFunc()
	res, err := plugin.cniConfig.AddNetwork(cniTimeoutCtx, config, rt)
	if err != nil {
		return nil, fmt.Errorf("error adding container to network: %v", err)
	}
	return res, nil
}

func (plugin *kubenetNetworkPlugin) delContainerFromNetwork(config *libcni.NetworkConfig, ifName, namespace, name string, id kubecontainer.ContainerID) error {
	rt, err := plugin.buildCNIRuntimeConf(ifName, id, false)
	if err != nil {
		return fmt.Errorf("error building CNI config: %v", err)
	}

	klog.V(3).InfoS("Removing pod from network with CNI plugin and runtime", "pod", klog.KRef(namespace, name), "networkName", config.Network.Name, "networkType", config.Network.Type, "rt", rt)
	// Because the default remote runtime request timeout is 4 min,so set slightly less than 240 seconds
	// Todo get the timeout from parent ctx
	cniTimeoutCtx, cancelFunc := context.WithTimeout(context.Background(), network.CNITimeoutSec*time.Second)
	defer cancelFunc()
	err = plugin.cniConfig.DelNetwork(cniTimeoutCtx, config, rt)
	// The pod may not get deleted successfully at the first time.
	// Ignore "no such file or directory" error in case the network has already been deleted in previous attempts.
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		return fmt.Errorf("error removing container from network: %v", err)
	}
	return nil
}

// shaper retrieves the bandwidth shaper and, if it hasn't been fetched before,
// initializes it and ensures the bridge is appropriately configured
func (plugin *kubenetNetworkPlugin) shaper() bandwidth.Shaper {
	plugin.mu.Lock()
	defer plugin.mu.Unlock()
	if plugin.bandwidthShaper == nil {
		plugin.bandwidthShaper = bandwidth.NewTCShaper(BridgeName)
		plugin.bandwidthShaper.ReconcileInterface()
	}
	return plugin.bandwidthShaper
}

//TODO: make this into a goroutine and rectify the dedup rules periodically
func (plugin *kubenetNetworkPlugin) syncEbtablesDedupRules(macAddr net.HardwareAddr, podCIDRs []net.IPNet, podGateways []net.IP) {
	if plugin.ebtables == nil {
		plugin.ebtables = utilebtables.New(plugin.execer)
		klog.V(3).InfoS("Flushing dedup chain")
		if err := plugin.ebtables.FlushChain(utilebtables.TableFilter, dedupChain); err != nil {
			klog.ErrorS(err, "Failed to flush dedup chain")
		}
	}
	_, err := plugin.ebtables.GetVersion()
	if err != nil {
		klog.InfoS("Failed to get ebtables version. Skip syncing ebtables dedup rules", "err", err)
		return
	}

	// ensure custom chain exists
	_, err = plugin.ebtables.EnsureChain(utilebtables.TableFilter, dedupChain)
	if err != nil {
		klog.ErrorS(nil, "Failed to ensure filter table KUBE-DEDUP chain")
		return
	}

	// jump to custom chain to the chain from core tables
	_, err = plugin.ebtables.EnsureRule(utilebtables.Append, utilebtables.TableFilter, utilebtables.ChainOutput, "-j", string(dedupChain))
	if err != nil {
		klog.ErrorS(err, "Failed to ensure filter table OUTPUT chain jump to KUBE-DEDUP chain")
		return
	}

	// per gateway rule
	for idx, gw := range podGateways {
		klog.V(3).InfoS("Filtering packets with ebtables", "mac", macAddr.String(), "gateway", gw.String(), "podCIDR", podCIDRs[idx].String())

		bIsV6 := netutils.IsIPv6(gw)
		IPFamily := "IPv4"
		ipSrc := "--ip-src"
		if bIsV6 {
			IPFamily = "IPv6"
			ipSrc = "--ip6-src"
		}
		commonArgs := []string{"-p", IPFamily, "-s", macAddr.String(), "-o", "veth+"}
		_, err = plugin.ebtables.EnsureRule(utilebtables.Prepend, utilebtables.TableFilter, dedupChain, append(commonArgs, ipSrc, gw.String(), "-j", "ACCEPT")...)
		if err != nil {
			klog.ErrorS(err, "Failed to ensure packets from cbr0 gateway to be accepted with error", "gateway", gw.String())
			return

		}
		_, err = plugin.ebtables.EnsureRule(utilebtables.Append, utilebtables.TableFilter, dedupChain, append(commonArgs, ipSrc, podCIDRs[idx].String(), "-j", "DROP")...)
		if err != nil {
			klog.ErrorS(err, "Failed to ensure packets from podCidr but has mac address of cbr0 to get dropped.", "podCIDR", podCIDRs[idx].String())
			return
		}
	}
}

// disableContainerDAD disables duplicate address detection in the container.
// DAD has a negative affect on pod creation latency, since we have to wait
// a second or more for the addresses to leave the "tentative" state. Since
// we're sure there won't be an address conflict (since we manage them manually),
// this is safe. See issue 54651.
//
// This sets net.ipv6.conf.default.dad_transmits to 0. It must be run *before*
// the CNI plugins are run.
func (plugin *kubenetNetworkPlugin) disableContainerDAD(id kubecontainer.ContainerID) error {
	key := "net/ipv6/conf/default/dad_transmits"

	sysctlBin, err := plugin.execer.LookPath("sysctl")
	if err != nil {
		return fmt.Errorf("could not find sysctl binary: %s", err)
	}

	netnsPath, err := plugin.host.GetNetNS(id.ID)
	if err != nil {
		return fmt.Errorf("failed to get netns: %v", err)
	}
	if netnsPath == "" {
		return fmt.Errorf("pod has no network namespace")
	}

	// If the sysctl doesn't exist, it means ipv6 is disabled; log and move on
	if _, err := plugin.sysctl.GetSysctl(key); err != nil {
		return fmt.Errorf("ipv6 not enabled: %v", err)
	}

	output, err := plugin.execer.Command(plugin.nsenterPath,
		fmt.Sprintf("--net=%s", netnsPath), "-F", "--",
		sysctlBin, "-w", fmt.Sprintf("%s=%s", key, "0"),
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to write sysctl: output: %s error: %s",
			output, err)
	}
	return nil
}

// given a n cidrs assigned to nodes,
// create bridge configuration that conforms to them
func (plugin *kubenetNetworkPlugin) getRangesConfig() string {
	createRange := func(thisNet *net.IPNet) string {
		template := `
[{
"subnet": "%s"
}]`
		return fmt.Sprintf(template, thisNet.String())
	}

	ranges := make([]string, len(plugin.podCIDRs))
	for idx, thisCIDR := range plugin.podCIDRs {
		ranges[idx] = createRange(thisCIDR)
	}
	//[{range}], [{range}]
	// each range contains a subnet. gateway will be fetched from cni result
	return strings.Join(ranges[:], ",")
}

// given a n cidrs assigned to nodes,
// create bridge routes configuration that conforms to them
func (plugin *kubenetNetworkPlugin) getRoutesConfig() string {
	var (
		routes       []string
		hasV4, hasV6 bool
	)
	for _, thisCIDR := range plugin.podCIDRs {
		if thisCIDR.IP.To4() != nil {
			hasV4 = true
		} else {
			hasV6 = true
		}
	}
	if hasV4 {
		routes = append(routes, fmt.Sprintf(`{"dst": "%s"}`, zeroCIDRv4))
	}
	if hasV6 {
		routes = append(routes, fmt.Sprintf(`{"dst": "%s"}`, zeroCIDRv6))
	}
	return strings.Join(routes, ",")
}

func (plugin *kubenetNetworkPlugin) addPodIP(id kubecontainer.ContainerID, ip string) {
	plugin.mu.Lock()
	defer plugin.mu.Unlock()

	_, exist := plugin.podIPs[id]
	if !exist {
		plugin.podIPs[id] = utilsets.NewString()
	}

	if !plugin.podIPs[id].Has(ip) {
		plugin.podIPs[id].Insert(ip)
	}
}

func (plugin *kubenetNetworkPlugin) removePodIP(id kubecontainer.ContainerID, ip string) {
	plugin.mu.Lock()
	defer plugin.mu.Unlock()

	_, exist := plugin.podIPs[id]
	if !exist {
		return // did we restart kubelet?
	}

	if plugin.podIPs[id].Has(ip) {
		plugin.podIPs[id].Delete(ip)
	}

	// if there is no more ips here. let us delete
	if plugin.podIPs[id].Len() == 0 {
		delete(plugin.podIPs, id)
	}
}

// returns a copy of pod ips
// false is returned if id does not exist
func (plugin *kubenetNetworkPlugin) getCachedPodIPs(id kubecontainer.ContainerID) ([]string, bool) {
	plugin.mu.Lock()
	defer plugin.mu.Unlock()

	iplist, exists := plugin.podIPs[id]
	if !exists {
		return nil, false
	}

	return iplist.UnsortedList(), true
}
