package net

import (
	"fmt"
	"os/exec"
	_net "net"
)

// IsIPV6 checks if the input contains a valid IPV6 address
func IsIPV6(ip _net.IP) bool {
	return ip != nil && ip.To4() == nil
}


// IsIPv6Enabled checks if IPV6 is enabled or not and we have
// at least one configured in the pod
func IsIPv6Enabled() bool {
	cmd := exec.Command("test", "-f", "/proc/net/if_inet6")
	if cmd.Run() != nil {
		return false
	}

	addrs, err := _net.InterfaceAddrs()
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		ip, _, _ := _net.ParseCIDR(addr.String())
		if IsIPV6(ip) {
			return true
		}
	}

	return false
}

// IsPortAvailable checks if a TCP port is available or not
func IsPortAvailable(p int) bool {
	conn, err := _net.Dial("tcp", fmt.Sprintf(":%v", p))
	if err != nil {
		return true
	}
	defer conn.Close()
	return false
}
