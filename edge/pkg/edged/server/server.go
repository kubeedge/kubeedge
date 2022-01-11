package server

import (
	"fmt"
	"k8s.io/kubernetes/pkg/kubelet/apis/podresources"
	"k8s.io/kubernetes/pkg/kubelet/util"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	podresourcesapi "k8s.io/kubelet/pkg/apis/podresources/v1"
	podresourcesapiv1alpha1 "k8s.io/kubelet/pkg/apis/podresources/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/apis/config"
	"k8s.io/kubernetes/pkg/kubelet/server"
	"k8s.io/kubernetes/pkg/kubelet/server/stats"
	"os"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/podmanager"
)

//constants to define server address
const (
	ServerAddr = "127.0.0.1"
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

// ListenAndServe starts a HTTP server and sets up a listener on the given host/port
func (s *Server) ListenAndServe(host server.HostInterface, resourceAnalyzer stats.ResourceAnalyzer, enableCAdvisorJSONEndpoints bool) {
	klog.Infof("starting to listen read-only on %s:%v", ServerAddr, constants.ServerPort)

	kubeCfg := &config.KubeletConfiguration{
		EnableDebuggingHandlers: true,
	}
	handler := server.NewServer(host, resourceAnalyzer, nil, kubeCfg)

	server := &http.Server{
		Addr:           net.JoinHostPort(ServerAddr, fmt.Sprintf("%d", constants.ServerPort)),
		Handler:        &handler,
		MaxHeaderBytes: 1 << 20,
	}
	klog.Exit(server.ListenAndServe())
}

// ListenAndServePodResources initializes a gRPC server to serve the PodResources service
func ListenAndServePodResources(socket string, podsProvider podresources.PodsProvider, devicesProvider podresources.DevicesProvider, cpusProvider podresources.CPUsProvider,  memoryProvider podresources.MemoryProvider) {
	server := grpc.NewServer()
	podresourcesapiv1alpha1.RegisterPodResourcesListerServer(server, podresources.NewV1alpha1PodResourcesServer(podsProvider, devicesProvider))
	podresourcesapi.RegisterPodResourcesListerServer(server, podresources.NewV1PodResourcesServer(podsProvider, devicesProvider, cpusProvider, memoryProvider))
	l, err := util.CreateListener(socket)
	if err != nil {
		klog.ErrorS(err, "Failed to create listener for podResources endpoint")
		os.Exit(1)
	}

	if err := server.Serve(l); err != nil {
		klog.ErrorS(err, "Failed to serve")
		os.Exit(1)
	}
}
