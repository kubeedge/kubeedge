// +build !dockerless

/*
Copyright 2017 The Kubernetes Authors.

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

package hostport

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	v1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	iptablesproxy "k8s.io/kubernetes/pkg/proxy/iptables"
	"k8s.io/kubernetes/pkg/util/conntrack"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	"k8s.io/utils/exec"
	utilnet "k8s.io/utils/net"
)

// HostPortManager is an interface for adding and removing hostport for a given pod sandbox.
type HostPortManager interface {
	// Add implements port mappings.
	// id should be a unique identifier for a pod, e.g. podSandboxID.
	// podPortMapping is the associated port mapping information for the pod.
	// natInterfaceName is the interface that localhost uses to talk to the given pod, if known.
	Add(id string, podPortMapping *PodPortMapping, natInterfaceName string) error
	// Remove cleans up matching port mappings
	// Remove must be able to clean up port mappings without pod IP
	Remove(id string, podPortMapping *PodPortMapping) error
}

type hostportManager struct {
	hostPortMap    map[hostport]closeable
	execer         exec.Interface
	conntrackFound bool
	iptables       utiliptables.Interface
	portOpener     hostportOpener
	mu             sync.Mutex
}

// NewHostportManager creates a new HostPortManager
func NewHostportManager(iptables utiliptables.Interface) HostPortManager {
	h := &hostportManager{
		hostPortMap: make(map[hostport]closeable),
		execer:      exec.New(),
		iptables:    iptables,
		portOpener:  openLocalPort,
	}
	h.conntrackFound = conntrack.Exists(h.execer)
	if !h.conntrackFound {
		klog.InfoS("The binary conntrack is not installed, this can cause failures in network connection cleanup.")
	}
	return h
}

func (hm *hostportManager) Add(id string, podPortMapping *PodPortMapping, natInterfaceName string) (err error) {
	if podPortMapping == nil || podPortMapping.HostNetwork {
		return nil
	}
	podFullName := getPodFullName(podPortMapping)
	// IP.To16() returns nil if IP is not a valid IPv4 or IPv6 address
	if podPortMapping.IP.To16() == nil {
		return fmt.Errorf("invalid or missing IP of pod %s", podFullName)
	}
	podIP := podPortMapping.IP.String()
	isIPv6 := utilnet.IsIPv6(podPortMapping.IP)

	// skip if there is no hostport needed
	hostportMappings := gatherHostportMappings(podPortMapping, isIPv6)
	if len(hostportMappings) == 0 {
		return nil
	}

	if isIPv6 != hm.iptables.IsIPv6() {
		return fmt.Errorf("HostPortManager IP family mismatch: %v, isIPv6 - %v", podIP, isIPv6)
	}

	if err := ensureKubeHostportChains(hm.iptables, natInterfaceName); err != nil {
		return err
	}

	// Ensure atomicity for port opening and iptables operations
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// try to open hostports
	ports, err := hm.openHostports(podPortMapping)
	if err != nil {
		return err
	}
	for hostport, socket := range ports {
		hm.hostPortMap[hostport] = socket
	}

	natChains := bytes.NewBuffer(nil)
	natRules := bytes.NewBuffer(nil)
	writeLine(natChains, "*nat")

	existingChains, existingRules, err := getExistingHostportIPTablesRules(hm.iptables)
	if err != nil {
		// clean up opened host port if encounter any error
		return utilerrors.NewAggregate([]error{err, hm.closeHostports(hostportMappings)})
	}

	newChains := []utiliptables.Chain{}
	conntrackPortsToRemove := []int{}
	for _, pm := range hostportMappings {
		protocol := strings.ToLower(string(pm.Protocol))
		chain := getHostportChain(id, pm)
		newChains = append(newChains, chain)
		if pm.Protocol == v1.ProtocolUDP {
			conntrackPortsToRemove = append(conntrackPortsToRemove, int(pm.HostPort))
		}

		// Add new hostport chain
		writeLine(natChains, utiliptables.MakeChainLine(chain))

		// Prepend the new chain to KUBE-HOSTPORTS
		// This avoids any leaking iptables rule that takes up the same port
		writeLine(natRules, "-I", string(kubeHostportsChain),
			"-m", "comment", "--comment", fmt.Sprintf(`"%s hostport %d"`, podFullName, pm.HostPort),
			"-m", protocol, "-p", protocol, "--dport", fmt.Sprintf("%d", pm.HostPort),
			"-j", string(chain),
		)

		// SNAT if the traffic comes from the pod itself
		writeLine(natRules, "-A", string(chain),
			"-m", "comment", "--comment", fmt.Sprintf(`"%s hostport %d"`, podFullName, pm.HostPort),
			"-s", podIP,
			"-j", string(iptablesproxy.KubeMarkMasqChain))

		// DNAT to the podIP:containerPort
		hostPortBinding := net.JoinHostPort(podIP, strconv.Itoa(int(pm.ContainerPort)))
		if pm.HostIP == "" || pm.HostIP == "0.0.0.0" || pm.HostIP == "::" {
			writeLine(natRules, "-A", string(chain),
				"-m", "comment", "--comment", fmt.Sprintf(`"%s hostport %d"`, podFullName, pm.HostPort),
				"-m", protocol, "-p", protocol,
				"-j", "DNAT", fmt.Sprintf("--to-destination=%s", hostPortBinding))
		} else {
			writeLine(natRules, "-A", string(chain),
				"-m", "comment", "--comment", fmt.Sprintf(`"%s hostport %d"`, podFullName, pm.HostPort),
				"-m", protocol, "-p", protocol, "-d", pm.HostIP,
				"-j", "DNAT", fmt.Sprintf("--to-destination=%s", hostPortBinding))
		}
	}

	// getHostportChain should be able to provide unique hostport chain name using hash
	// if there is a chain conflict or multiple Adds have been triggered for a single pod,
	// filtering should be able to avoid further problem
	filterChains(existingChains, newChains)
	existingRules = filterRules(existingRules, newChains)

	for _, chain := range existingChains {
		writeLine(natChains, chain)
	}
	for _, rule := range existingRules {
		writeLine(natRules, rule)
	}
	writeLine(natRules, "COMMIT")

	if err = hm.syncIPTables(append(natChains.Bytes(), natRules.Bytes()...)); err != nil {
		// clean up opened host port if encounter any error
		return utilerrors.NewAggregate([]error{err, hm.closeHostports(hostportMappings)})
	}

	// Remove conntrack entries just after adding the new iptables rules. If the conntrack entry is removed along with
	// the IP tables rule, it can be the case that the packets received by the node after iptables rule removal will
	// create a new conntrack entry without any DNAT. That will result in blackhole of the traffic even after correct
	// iptables rules have been added back.
	if hm.execer != nil && hm.conntrackFound {
		klog.InfoS("Starting to delete udp conntrack entries", "conntrackEntries", conntrackPortsToRemove, "isIPv6", isIPv6)
		for _, port := range conntrackPortsToRemove {
			err = conntrack.ClearEntriesForPort(hm.execer, port, isIPv6, v1.ProtocolUDP)
			if err != nil {
				klog.ErrorS(err, "Failed to clear udp conntrack for port", "port", port)
			}
		}
	}
	return nil
}

func (hm *hostportManager) Remove(id string, podPortMapping *PodPortMapping) (err error) {
	if podPortMapping == nil || podPortMapping.HostNetwork {
		return nil
	}

	hostportMappings := gatherHostportMappings(podPortMapping, hm.iptables.IsIPv6())
	if len(hostportMappings) == 0 {
		return nil
	}

	// Ensure atomicity for port closing and iptables operations
	hm.mu.Lock()
	defer hm.mu.Unlock()

	var existingChains map[utiliptables.Chain]string
	var existingRules []string
	existingChains, existingRules, err = getExistingHostportIPTablesRules(hm.iptables)
	if err != nil {
		return err
	}

	// Gather target hostport chains for removal
	chainsToRemove := []utiliptables.Chain{}
	for _, pm := range hostportMappings {
		chainsToRemove = append(chainsToRemove, getHostportChain(id, pm))
	}

	// remove rules that consists of target chains
	remainingRules := filterRules(existingRules, chainsToRemove)

	// gather target hostport chains that exists in iptables-save result
	existingChainsToRemove := []utiliptables.Chain{}
	for _, chain := range chainsToRemove {
		if _, ok := existingChains[chain]; ok {
			existingChainsToRemove = append(existingChainsToRemove, chain)
		}
	}

	// exit if there is nothing to remove
	// don´t forget to clean up opened pod host ports
	if len(existingChainsToRemove) == 0 {
		return hm.closeHostports(hostportMappings)
	}

	natChains := bytes.NewBuffer(nil)
	natRules := bytes.NewBuffer(nil)
	writeLine(natChains, "*nat")
	for _, chain := range existingChains {
		writeLine(natChains, chain)
	}
	for _, rule := range remainingRules {
		writeLine(natRules, rule)
	}
	for _, chain := range existingChainsToRemove {
		writeLine(natRules, "-X", string(chain))
	}
	writeLine(natRules, "COMMIT")

	if err := hm.syncIPTables(append(natChains.Bytes(), natRules.Bytes()...)); err != nil {
		return err
	}

	// clean up opened pod host ports
	return hm.closeHostports(hostportMappings)
}

// syncIPTables executes iptables-restore with given lines
func (hm *hostportManager) syncIPTables(lines []byte) error {
	klog.V(3).InfoS("Restoring iptables rules", "iptableRules", lines)
	err := hm.iptables.RestoreAll(lines, utiliptables.NoFlushTables, utiliptables.RestoreCounters)
	if err != nil {
		return fmt.Errorf("failed to execute iptables-restore: %v", err)
	}
	return nil
}

// openHostports opens all given hostports using the given hostportOpener
// If encounter any error, clean up and return the error
// If all ports are opened successfully, return the hostport and socket mapping
func (hm *hostportManager) openHostports(podPortMapping *PodPortMapping) (map[hostport]closeable, error) {
	var retErr error
	ports := make(map[hostport]closeable)
	for _, pm := range podPortMapping.PortMappings {
		if pm.HostPort <= 0 {
			continue
		}

		// We do not open host ports for SCTP ports, as we agreed in the Support of SCTP KEP
		if pm.Protocol == v1.ProtocolSCTP {
			continue
		}

		// HostIP IP family is not handled by this port opener
		if pm.HostIP != "" && utilnet.IsIPv6String(pm.HostIP) != hm.iptables.IsIPv6() {
			continue
		}

		hp := portMappingToHostport(pm, hm.getIPFamily())
		socket, err := hm.portOpener(&hp)
		if err != nil {
			retErr = fmt.Errorf("cannot open hostport %d for pod %s: %v", pm.HostPort, getPodFullName(podPortMapping), err)
			break
		}
		ports[hp] = socket
	}

	// If encounter any error, close all hostports that just got opened.
	if retErr != nil {
		for hp, socket := range ports {
			if err := socket.Close(); err != nil {
				klog.ErrorS(err, "Cannot clean up hostport for the pod", "podFullName", getPodFullName(podPortMapping), "port", hp.port)
			}
		}
		return nil, retErr
	}
	return ports, nil
}

// closeHostports tries to close all the listed host ports
func (hm *hostportManager) closeHostports(hostportMappings []*PortMapping) error {
	errList := []error{}
	for _, pm := range hostportMappings {
		hp := portMappingToHostport(pm, hm.getIPFamily())
		if socket, ok := hm.hostPortMap[hp]; ok {
			klog.V(2).InfoS("Closing host port", "port", hp.String())
			if err := socket.Close(); err != nil {
				errList = append(errList, fmt.Errorf("failed to close host port %s: %v", hp.String(), err))
				continue
			}
			delete(hm.hostPortMap, hp)
		} else {
			klog.V(5).InfoS("Host port does not have an open socket", "port", hp.String())
		}
	}
	return utilerrors.NewAggregate(errList)
}

// getIPFamily returns the hostPortManager IP family
func (hm *hostportManager) getIPFamily() ipFamily {
	family := IPv4
	if hm.iptables.IsIPv6() {
		family = IPv6
	}
	return family
}

// getHostportChain takes id, hostport and protocol for a pod and returns associated iptables chain.
// This is computed by hashing (sha256) then encoding to base32 and truncating with the prefix
// "KUBE-HP-". We do this because IPTables Chain Names must be <= 28 chars long, and the longer
// they are the harder they are to read.
// WARNING: Please do not change this function. Otherwise, HostportManager may not be able to
// identify existing iptables chains.
func getHostportChain(id string, pm *PortMapping) utiliptables.Chain {
	hash := sha256.Sum256([]byte(id + strconv.Itoa(int(pm.HostPort)) + string(pm.Protocol) + pm.HostIP))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return utiliptables.Chain(kubeHostportChainPrefix + encoded[:16])
}

// gatherHostportMappings returns all the PortMappings which has hostport for a pod
// it filters the PortMappings that use HostIP and doesn't match the IP family specified
func gatherHostportMappings(podPortMapping *PodPortMapping, isIPv6 bool) []*PortMapping {
	mappings := []*PortMapping{}
	for _, pm := range podPortMapping.PortMappings {
		if pm.HostPort <= 0 {
			continue
		}
		if pm.HostIP != "" && utilnet.IsIPv6String(pm.HostIP) != isIPv6 {
			continue
		}
		mappings = append(mappings, pm)
	}
	return mappings
}

// getExistingHostportIPTablesRules retrieves raw data from iptables-save, parse it,
// return all the hostport related chains and rules
func getExistingHostportIPTablesRules(iptables utiliptables.Interface) (map[utiliptables.Chain]string, []string, error) {
	iptablesData := bytes.NewBuffer(nil)
	err := iptables.SaveInto(utiliptables.TableNAT, iptablesData)
	if err != nil { // if we failed to get any rules
		return nil, nil, fmt.Errorf("failed to execute iptables-save: %v", err)
	}
	existingNATChains := utiliptables.GetChainLines(utiliptables.TableNAT, iptablesData.Bytes())

	existingHostportChains := make(map[utiliptables.Chain]string)
	existingHostportRules := []string{}

	for chain := range existingNATChains {
		if strings.HasPrefix(string(chain), string(kubeHostportsChain)) || strings.HasPrefix(string(chain), kubeHostportChainPrefix) {
			existingHostportChains[chain] = string(existingNATChains[chain])
		}
	}

	for _, line := range strings.Split(iptablesData.String(), "\n") {
		if strings.HasPrefix(line, fmt.Sprintf("-A %s", kubeHostportChainPrefix)) ||
			strings.HasPrefix(line, fmt.Sprintf("-A %s", string(kubeHostportsChain))) {
			existingHostportRules = append(existingHostportRules, line)
		}
	}
	return existingHostportChains, existingHostportRules, nil
}

// filterRules filters input rules with input chains. Rules that did not involve any filter chain will be returned.
// The order of the input rules is important and is preserved.
func filterRules(rules []string, filters []utiliptables.Chain) []string {
	filtered := []string{}
	for _, rule := range rules {
		skip := false
		for _, filter := range filters {
			if strings.Contains(rule, string(filter)) {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

// filterChains deletes all entries of filter chains from chain map
func filterChains(chains map[utiliptables.Chain]string, filterChains []utiliptables.Chain) {
	for _, chain := range filterChains {
		delete(chains, chain)
	}
}

func getPodFullName(pod *PodPortMapping) string {
	// Use underscore as the delimiter because it is not allowed in pod name
	// (DNS subdomain format), while allowed in the container name format.
	return pod.Name + "_" + pod.Namespace
}

// Join all words with spaces, terminate with newline and write to buf.
func writeLine(buf *bytes.Buffer, words ...string) {
	buf.WriteString(strings.Join(words, " ") + "\n")
}

func (hp *hostport) String() string {
	return fmt.Sprintf("%s:%d", hp.protocol, hp.port)
}
