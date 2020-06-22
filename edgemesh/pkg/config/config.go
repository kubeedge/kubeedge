package config

import (
	"net"
	"sync"

	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgeMesh
	// for edgemesh listener
	ListenIP net.IP
	Listener *net.TCPListener
}

func InitConfigure(e *v1alpha1.EdgeMesh) {
	once.Do(func() {
		Config = Configure{
			EdgeMesh: *e,
		}
		if Config.Enable {
			Config.ListenIP = net.ParseIP(e.InterfaceAddr)
			// get listener
			tmpPort := 0
			listenAddr := &net.TCPAddr{
				IP:   Config.ListenIP,
				Port: Config.ListenPort + tmpPort,
			}
			for {
				ln, err := net.ListenTCP("tcp", listenAddr)
				if err == nil {
					Config.Listener = ln
					break
				}
				klog.Warningf("[EdgeMesh] listen on address %v err: %v", listenAddr, err)
				tmpPort++
				listenAddr = &net.TCPAddr{
					IP:   Config.ListenIP,
					Port: Config.ListenPort + tmpPort,
				}
			}
		}
	})
}
