package dns

import (
	"io/ioutil"
	"k8s.io/klog/v2"
	"net"
	"strings"
)

var defResolvConf = "/etc/resolv.conf"

// GetSystemNameServers returns the list of nameservers located in the file /etc/resolv.conf
func GetSystemNameServers() ([]net.IP, error) {
	var nameservers []net.IP
	file, err := ioutil.ReadFile(defResolvConf)
	if err != nil {
		return nameservers, err
	}

	// Lines of the form "nameserver 1.2.3.4" accumulate.
	lines := strings.Split(string(file), "\n")
	for l := range lines {
		trimmed := strings.TrimSpace(lines[l])
		if len(trimmed) == 0 || trimmed[0] == '#' || trimmed[0] == ';' {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == "nameserver" {
			ip := net.ParseIP(fields[1])
			if ip != nil {
				nameservers = append(nameservers, ip)
			}
		}
	}

	klog.V(3).Info("Nameservers", "hosts", nameservers)
	return nameservers, nil
}