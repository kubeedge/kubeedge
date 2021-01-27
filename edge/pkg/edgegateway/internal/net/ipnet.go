package net

import (
	"net"
	"strings"
)

// IPNet maps string to net.IPNet.
type IPNet map[string]*net.IPNet

// IP maps string to net.IP.
type IP map[string]net.IP

// ParseIPNets parses string slice to IPNet.
func ParseIPNets(specs ...string) (IPNet, IP, error) {
	ipnetset := make(IPNet)
	ipset := make(IP)

	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		_, ipnet, err := net.ParseCIDR(spec)
		if err != nil {
			ip := net.ParseIP(spec)
			if ip == nil {
				return nil, nil, err
			}
			i := ip.String()
			ipset[i] = ip
			continue
		}

		k := ipnet.String()
		ipnetset[k] = ipnet
	}

	return ipnetset, ipset, nil
}