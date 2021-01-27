package edgegateway

import (
	"fmt"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	fakekube "github.com/kubeedge/kubeedge/edge/pkg/edged/fake"
	gatewayconfig "github.com/kubeedge/kubeedge/edge/pkg/edgegateway/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/ingress/controller"
	"github.com/kubeedge/kubeedge/edge/pkg/edgegateway/internal/nginx"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"k8s.io/apiserver/pkg/server/healthz"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// edgeGateway struct
type edgeGateway struct {
	kubeClient clientset.Interface
	metaClient client.CoreInterface
	discovery discovery
	enable bool
}

func newEdgeGateway(enable bool) *edgeGateway {
	// create metaManager client
	metaClient := client.New()

	return &edgeGateway{
		metaClient: metaClient,
		kubeClient: fakekube.NewSimpleClientset(metaClient),
		enable: enable,
	}
}

// Register register edgeGateway
func Register(edgeGateway *v1alpha1.EdgeGateway,nodeName string)  {
	gatewayconfig.InitConfigure(edgeGateway,nodeName)
	core.Register(newEdgeGateway(edgeGateway.Enable))
}

//Name returns the name of EdgeGateway module
func (e *edgeGateway) Name() string {
	return "edgeGateway"
}

//Group returns EdgeGateway group
func (e *edgeGateway) Group() string {
	return modules.GatewayGroup
}

// Enable indicates whether this module is enabled
func (e *edgeGateway) Enable() bool {
	return e.enable
}

//Start sets context and starts the controller
func (e *edgeGateway) Start() {
	klog.Infof("Starting EdgeGateway Server with edge")

	metaClient := client.New()
	kubeClient := fakekube.NewSimpleClientset(metaClient)

	if kubeClient == nil {
		klog.Fatalf("change metaClient to kubeClient error")
	}

	// start edge discovery
	err,conf := e.discovery.Start(kubeClient)

	if err != nil {
		klog.Fatalf("EdgeGateway discovery server has error %v", err)
	}

	// Proxy API interface
	klog.Infof("Starting proxy API server")
	err = e.Proxy(conf)
	if err != nil {
		klog.Fatalf("EdgeGateway Proxy API server has error %v", err)
	}

}

// Proxy
func (e *edgeGateway) Proxy(conf *controller.Configuration) (err error) {

	// new nginx controller
	ngx := controller.NewNginxController(conf)

	mux := http.NewServeMux()
	registerHealthz(nginx.HealthPath, ngx, mux)

	// start HTTP Server
	go startHTTPServer(conf.ListenPorts.Health, mux)

	// start nginx ingress controller
	go ngx.Start()

	handleSigterm(ngx, func(code int) {
		os.Exit(code)
	})

	return err
}

type exiter func(code int)

func handleSigterm(ngx *controller.NginxController, exit exiter) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan
	klog.Info("Received SIGTERM, shutting down")

	exitCode := 0
	if err := ngx.Stop(); err != nil {
		klog.Warningf("Error during shutdown: %v", err)
		exitCode = 1
	}

	klog.Info("Handled quit, awaiting Pod deletion")
	time.Sleep(10 * time.Second)

	klog.Info("Exiting", "code", exitCode)
	exit(exitCode)
}
func registerHealthz(healthPath string, ic *controller.NginxController, mux *http.ServeMux) {
	// expose health check endpoint (/healthz)
	healthz.InstallPathHandler(mux,
		healthPath,
		healthz.PingHealthz,
		ic,
	)
}

func registerProfiler() {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/heap", pprof.Index)
	mux.HandleFunc("/debug/pprof/mutex", pprof.Index)
	mux.HandleFunc("/debug/pprof/goroutine", pprof.Index)
	mux.HandleFunc("/debug/pprof/threadcreate", pprof.Index)
	mux.HandleFunc("/debug/pprof/block", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%v", nginx.ProfilerPort),
		Handler: mux,
	}
	klog.Fatal(server.ListenAndServe())
}

func startHTTPServer(port int, mux *http.ServeMux) {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%v", port),
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      300 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	klog.Fatal(server.ListenAndServe())
}
