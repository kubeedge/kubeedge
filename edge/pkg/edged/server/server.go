package server

import (
	"fmt"
	"net"
	"net/http"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/config"

	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubelet/server"
	"k8s.io/kubernetes/pkg/kubelet/server/stats"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/podmanager"
)

//constants to define server address
const (
	ServerAddr = "127.0.0.1"
)

//TunnelServer is object to define server
type Server struct {
	podManager podmanager.Manager
}

//NewServer creates and returns a new server object
func NewServer(podManager podmanager.Manager) *Server {
	return &Server{
		podManager: podManager,
	}
}

// ListenAndServe starts a HTTP server and sets up a listener on the given host/port
func (s *Server) ListenAndServe(host server.HostInterface, resourceAnalyzer stats.ResourceAnalyzer, enableCAdvisorJSONEndpoints bool) {
	klog.Infof("starting to listen read-only on %s:%v", ServerAddr, config.ServerPort)
	handler := server.NewServer(host, resourceAnalyzer, nil, enableCAdvisorJSONEndpoints, true, false, false, nil)

	server := &http.Server{
		Addr:           net.JoinHostPort(ServerAddr, fmt.Sprintf("%d", config.ServerPort)),
		Handler:        &handler,
		MaxHeaderBytes: 1 << 20,
	}
	klog.Fatal(server.ListenAndServe())
}
