/*
Copyright 2020 The KubeEdge Authors.

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

package config

import (
	"net"
	"sync"

	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edgemesh/pkg/common"
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
			// get listen ip
			var err error
			Config.ListenIP, err = common.GetInterfaceIP(Config.ListenInterface)
			if err != nil {
				klog.Errorf("[EdgeMesh] get listen ip err: %v", err)
				return
			}
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
