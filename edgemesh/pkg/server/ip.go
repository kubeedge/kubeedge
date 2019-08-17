package server

import (
	"net"
	"time"

	"k8s.io/klog"
)

const inter = "docker0"

// getIP returns the specific interface ip of version 4
func getIP() (net.IP, error) {
	for {
		ifaces, err := net.InterfaceByName(inter)
		if err != nil {
			return nil, err
		}
		addrs, _ := ifaces.Addrs()
		for _, addr := range addrs {
			if ip, inet, _ := net.ParseCIDR(addr.String()); len(inet.Mask) == 4 {
				return ip, nil
			}
		}
		klog.Warningf("the interface %s have not config ip of version 4", inter)
		time.Sleep(time.Second * 3)
	}
}
