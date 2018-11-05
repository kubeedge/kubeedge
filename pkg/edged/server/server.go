package server

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"edge-core/beehive/pkg/common/log"
	"edge-core/pkg/edged/podmanager"
	"k8s.io/api/core/v1"
)

const (
	NET_INTERFACE = "eth0"
	SERVER_ADDR   = "127.0.0.1"
	SERVER_PORT   = 10255
)

type Server struct {
	podManager podmanager.Manager
}

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

func getLocalIp() string {
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

func (s *Server) ListenAndServe() {
	//addr := getLocalIp()
	log.LOGGER.Infof("starting to listen on %s:%d", SERVER_ADDR, SERVER_PORT)
	mux := http.NewServeMux()
	mux.HandleFunc("/pods", s.getPodsHandler)
	err := http.ListenAndServe(net.JoinHostPort(SERVER_ADDR, strconv.FormatUint(uint64(SERVER_PORT), 10)), mux)
	if err != nil {
		log.LOGGER.Fatalf("run server: %v", err)
	}
}
