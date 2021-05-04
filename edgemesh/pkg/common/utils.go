package common

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/go-chassis/go-chassis/core/common"
)

func SplitServiceKey(key string) (name, namespace string) {
	sets := strings.Split(key, ".")
	if len(sets) >= 2 {
		return sets[0], sets[1]
	}

	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		ns = common.DefaultValue
	}
	if len(sets) == 1 {
		return sets[0], ns
	}
	return key, ns
}

func GetInterfaceIP(name string) (net.IP, error) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	addrs, _ := ifi.Addrs()
	for _, addr := range addrs {
		if ip, ipn, _ := net.ParseCIDR(addr.String()); len(ipn.Mask) == 4 {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no ip of version 4 found for interface %s", name)
}
