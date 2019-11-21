package server

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"k8s.io/api/core/v1"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/podmanager"
)

//constants to define server address
const (
	ServerAddr = "127.0.0.1"
	ServerPort = 10255
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

// ListenAndServe starts a HTTP server and sets up a listener on the given host/port
func (s *Server) ListenAndServe() {
	klog.Infof("starting to listen on %s:%d", ServerAddr, ServerPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/pods", s.getPodsHandler)
	err := http.ListenAndServe(net.JoinHostPort(ServerAddr, strconv.FormatUint(uint64(ServerPort), 10)), mux)
	if err != nil {
		klog.Fatalf("run server: %v", err)
	}
}
