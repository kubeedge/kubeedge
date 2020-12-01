package proxy

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/edgemesh/pkg/config"
)

// iptables rules
type Proxier struct {
	iptables     utiliptables.Interface
	inboundRule  string
	outboundRule string
	dNatRule     string
}

const (
	meshChain  = "EDGE-MESH"
	hostResolv = "/etc/resolv.conf"
)

var (
	proxier *Proxier
	route   netlink.Route
)

func Init() {
	protocol := utiliptables.ProtocolIPv4
	exec := utilexec.New()
	iptInterface := utiliptables.New(exec, protocol)
	proxier = &Proxier{
		iptables:     iptInterface,
		inboundRule:  "-p tcp -d " + config.Config.SubNet + " -i " + config.Config.ListenInterface + " -j " + meshChain,
		outboundRule: "-p tcp -d " + config.Config.SubNet + " -o " + config.Config.ListenInterface + " -j " + meshChain,
		dNatRule:     "-p tcp -j DNAT --to-destination " + config.Config.Listener.Addr().String(),
	}
	// read and clean iptables rules
	proxier.readAndCleanRule()
	// ensure iptables rules
	proxier.ensureRule()
	// add route
	dst, err := netlink.ParseIPNet(config.Config.SubNet)
	if err != nil {
		klog.Errorf("[EdgeMesh] parse subnet error: %v", err)
		return
	}
	gw := config.Config.ListenIP
	route = netlink.Route{
		Dst: dst,
		Gw:  gw,
	}
	err = netlink.RouteAdd(&route)
	if err != nil {
		klog.Warningf("[EdgeMesh] add route err: %v", err)
	}
	// save iptables rules
	proxier.saveRule()
	// ensure resolv.conf
	ensureResolvForHost()
	// sync
	go proxier.sync()
}

// sync periodically
func (p *Proxier) sync() {
	syncRuleTicker := time.NewTicker(10 * time.Second)
	for {
		<-syncRuleTicker.C
		p.ensureRule()
		ensureResolvForHost()
	}
}

// ensureRule ensures iptables rules exist
func (p *Proxier) ensureRule() {
	iptInterface := p.iptables
	inboundRule := strings.Split(p.inboundRule, " ")
	outboundRule := strings.Split(p.outboundRule, " ")
	dNatRule := strings.Split(p.dNatRule, " ")
	exist, err := iptInterface.EnsureChain(utiliptables.TableNAT, meshChain)
	if err != nil {
		klog.Errorf("[EdgeMesh] ensure chain %s failed with err: %v", meshChain, err)
	}
	if !exist {
		klog.Infof("[EdgeMesh] chain %s not exists", meshChain)
	}

	exist, err = iptInterface.EnsureRule(utiliptables.Append, utiliptables.TableNAT, utiliptables.ChainPrerouting, inboundRule...)
	if err != nil {
		klog.Errorf("[EdgeMesh] ensure inbound rule %s failed with err: %v", p.inboundRule, err)
	}
	if !exist {
		klog.Infof("[EdgeMesh] inbound rule %s not exists", p.inboundRule)
	}

	exist, err = iptInterface.EnsureRule(utiliptables.Append, utiliptables.TableNAT, utiliptables.ChainOutput, outboundRule...)
	if err != nil {
		klog.Errorf("[EdgeMesh] ensure outbound rule %s failed with err: %v", p.outboundRule, err)
	}
	if !exist {
		klog.Infof("[EdgeMesh] outbound rule %s not exists", p.outboundRule)
	}

	exist, err = iptInterface.EnsureRule(utiliptables.Append, utiliptables.TableNAT, meshChain, dNatRule...)
	if err != nil {
		klog.Errorf("[EdgeMesh] ensure dnat rule %s failed with err: %v", p.dNatRule, err)
	}
	if !exist {
		klog.Infof("[EdgeMesh] dnat rule %s not exists", p.dNatRule)
	}
}

// saveRule saves iptables rules into file
func (p *Proxier) saveRule() {
	file, err := os.OpenFile("/run/edgemesh-iptables", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		klog.Errorf("[EdgeMesh] open file /run/edgemesh-iptables err: %v", err)
		return
	}
	// store
	defer file.Close()
	w := bufio.NewWriter(file)
	fmt.Fprintln(w, p.inboundRule)
	fmt.Fprintln(w, p.dNatRule)
	fmt.Fprintln(w, p.outboundRule)
	w.Flush()
}

// readAndCleanRule reads iptables rules from file and cleans them
func (p *Proxier) readAndCleanRule() {
	file, err := os.OpenFile("/run/edgemesh-iptables", os.O_RDONLY, 0444)
	if err != nil {
		klog.Errorf("[EdgeMesh] open file /run/edgemesh-iptables err: %v", err)
		return
	}

	defer file.Close()
	scan := bufio.NewScanner(file)
	scan.Split(bufio.ScanLines)
	for scan.Scan() {
		serverString := scan.Text()
		if strings.Contains(serverString, "-o") {
			if err := p.iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainOutput, strings.Split(serverString, " ")...); err != nil {
				klog.Errorf("[EdgeMesh] failed to delete iptables rule, err: %v", err)
				return
			}
		} else if strings.Contains(serverString, "-i") {
			if err := p.iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainPrerouting, strings.Split(serverString, " ")...); err != nil {
				klog.Errorf("[EdgeMesh] failed to delete iptables rule, err: %v", err)
				return
			}
		}
	}
	if err := p.iptables.FlushChain(utiliptables.TableNAT, meshChain); err != nil {
		klog.Errorf("[EdgeMesh] failed to flush iptables chain, err: %v", err)
		return
	}
	if err := p.iptables.DeleteChain(utiliptables.TableNAT, meshChain); err != nil {
		klog.Errorf("[EdgeMesh] failed to delete iptables chain, err: %v", err)
		return
	}
}

// ensureResolvForHost adds edgemesh dns server to the head of /etc/resolv.conf
func ensureResolvForHost() {
	bs, err := ioutil.ReadFile(hostResolv)
	if err != nil {
		klog.Errorf("[EdgeMesh] read file %s err: %v", hostResolv, err)
		return
	}

	resolv := strings.Split(string(bs), "\n")
	if resolv == nil {
		nameserver := "nameserver " + config.Config.ListenIP.String()
		if err := ioutil.WriteFile(hostResolv, []byte(nameserver), 0600); err != nil {
			klog.Errorf("[EdgeMesh] write file %s err: %v", hostResolv, err)
		}
		return
	}

	configured := false
	dnsIdx := 0
	startIdx := 0
	for idx, item := range resolv {
		if strings.Contains(item, config.Config.ListenIP.String()) {
			configured = true
			dnsIdx = idx
			break
		}
	}
	for idx, item := range resolv {
		if strings.Contains(item, "nameserver") {
			startIdx = idx
			break
		}
	}
	if configured {
		if dnsIdx != startIdx && dnsIdx > startIdx {
			nameserver := sortNameserver(resolv, dnsIdx, startIdx)
			if err := ioutil.WriteFile(hostResolv, []byte(nameserver), 0600); err != nil {
				klog.Errorf("[EdgeMesh] failed to write file %s, err: %v", hostResolv, err)
				return
			}
		}
		return
	}

	nameserver := ""
	for idx := 0; idx < len(resolv); {
		if idx == startIdx {
			startIdx = -1
			nameserver = nameserver + "nameserver " + config.Config.ListenIP.String() + "\n"
			continue
		}
		nameserver = nameserver + resolv[idx] + "\n"
		idx++
	}

	if err := ioutil.WriteFile(hostResolv, []byte(nameserver), 0600); err != nil {
		klog.Errorf("[EdgeMesh] failed to write file %s, err: %v", hostResolv, err)
		return
	}
}

func sortNameserver(resolv []string, dnsIdx, startIdx int) string {
	nameserver := ""
	idx := 0
	for ; idx < startIdx; idx++ {
		nameserver = nameserver + resolv[idx] + "\n"
	}
	nameserver = nameserver + resolv[dnsIdx] + "\n"

	for idx = startIdx; idx < len(resolv); idx++ {
		if idx == dnsIdx {
			continue
		}
		nameserver = nameserver + resolv[idx] + "\n"
	}

	return nameserver
}

func Clean() {
	proxier.readAndCleanRule()
	if err := netlink.RouteDel(&route); err != nil {
		klog.Warningf("[EdgeMesh] delete route err: %v", err)
	}
	bs, err := ioutil.ReadFile(hostResolv)
	if err != nil {
		klog.Warningf("[EdgeMesh] read file %s err: %v", hostResolv, err)
	}

	resolv := strings.Split(string(bs), "\n")
	if resolv == nil {
		return
	}
	nameserver := ""
	for _, item := range resolv {
		if strings.Contains(item, config.Config.ListenIP.String()) {
			continue
		}
		nameserver = nameserver + item + "\n"
	}
	if err := ioutil.WriteFile(hostResolv, []byte(nameserver), 0600); err != nil {
		klog.Errorf("[EdgeMesh] failed to write nameserver to file %s, err: %v", hostResolv, err)
	}
}
