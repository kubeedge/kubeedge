/*
Copyright 2019 The KubeEdge Authors.

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
package server

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/podmanager"

	"k8s.io/api/core/v1"
)

//constants to define server address
const (
	NetInterface = "eth0"
	ServerAddr   = "127.0.0.1"
	ServerPort   = 10255
)

//Server is object to define server
type Server struct {
	podManager podmanager.Manager
}

//NewServer creates and returns a new server object
func NewServer(podManager podmanager.Manager) *Server {
	return &Server{
		podManager: podManager,
	}
}

func (s *Server) getPodsHandler(w http.ResponseWriter, r *http.Request) {
	var podList v1.PodList
	pods := s.podManager.GetPods()
	for _, pod := range pods {
		podList.Items = append(podList.Items, *pod)
	}
	rspBodyBytes := new(bytes.Buffer)
	json.NewEncoder(rspBodyBytes).Encode(podList)
	w.Write(rspBodyBytes.Bytes())
}

func getLocalIP() string {
	addrSlice, err := net.InterfaceAddrs()
	if nil != err {
		log.LOGGER.Errorf("Get local IP addr failed %s", err.Error())
		return "localhost"
	}
	for _, addr := range addrSlice {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if nil != ipnet.IP.To4() {
				return ipnet.IP.String()
			}
		}
	}
	return "localhost"
}

// ListenAndServe starts a HTTP server and sets up a listener on the given host/port
func (s *Server) ListenAndServe() {
	//addr := getLocalIp()
	log.LOGGER.Infof("starting to listen on %s:%d", ServerAddr, ServerPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/pods", s.getPodsHandler)
	err := http.ListenAndServe(net.JoinHostPort(ServerAddr, strconv.FormatUint(uint64(ServerPort), 10)), mux)
	if err != nil {
		log.LOGGER.Fatalf("run server: %v", err)
	}
}
