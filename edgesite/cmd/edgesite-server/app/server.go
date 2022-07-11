package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/apiserver-network-proxy/cmd/server/app/options"
	"sigs.k8s.io/apiserver-network-proxy/konnectivity-client/proto/client"
	"sigs.k8s.io/apiserver-network-proxy/pkg/server"
	"sigs.k8s.io/apiserver-network-proxy/pkg/util"
	"sigs.k8s.io/apiserver-network-proxy/proto/agent"
)

const grpcMode = "grpc"

var udsListenerLock sync.Mutex

func NewProxyCommand(p *Proxy, o *options.ProxyRunOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "proxy",
		Long: `A gRPC proxy server, receives requests from the API server and forwards to the agent.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return p.run(o)
		},
	}

	return cmd
}

type Proxy struct {
}

type StopFunc func()

func (p *Proxy) run(o *options.ProxyRunOptions) error {
	o.Print()
	if err := o.Validate(); err != nil {
		return fmt.Errorf("failed to validate server options with %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var k8sClient *kubernetes.Clientset
	if o.AgentNamespace != "" {
		config, err := clientcmd.BuildConfigFromFlags("", o.KubeconfigPath)
		if err != nil {
			return fmt.Errorf("failed to load kubernetes client config: %v", err)
		}

		if o.KubeconfigQPS != 0 {
			klog.V(1).Infof("Setting k8s client QPS: %v", o.KubeconfigQPS)
			config.QPS = o.KubeconfigQPS
		}
		if o.KubeconfigBurst != 0 {
			klog.V(1).Infof("Setting k8s client Burst: %v", o.KubeconfigBurst)
			config.Burst = o.KubeconfigBurst
		}
		k8sClient, err = kubernetes.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("failed to create kubernetes clientset: %v", err)
		}
	}

	authOpt := &server.AgentTokenAuthenticationOptions{
		Enabled:                o.AgentNamespace != "",
		AgentNamespace:         o.AgentNamespace,
		AgentServiceAccount:    o.AgentServiceAccount,
		KubernetesClient:       k8sClient,
		AuthenticationAudience: o.AuthenticationAudience,
	}
	klog.V(1).Infoln("Starting master server for client connections.")
	ps, err := server.GenProxyStrategiesFromStr(o.ProxyStrategies)
	if err != nil {
		return err
	}
	server := server.NewProxyServer(o.ServerID, ps, int(o.ServerCount), authOpt, true)

	masterStop, err := p.runMasterServer(ctx, o, server)
	if err != nil {
		return fmt.Errorf("failed to run the master server: %v", err)
	}

	klog.V(1).Infoln("Starting agent server for tunnel connections.")
	err = p.runAgentServer(o, server)
	if err != nil {
		return fmt.Errorf("failed to run the agent server: %v", err)
	}
	klog.V(1).Infoln("Starting admin server for debug connections.")
	p.runAdminServer(o)

	klog.V(1).Infoln("Starting health server for healthchecks.")
	p.runHealthServer(o, server)

	stopCh := SetupSignalHandler()
	<-stopCh
	klog.V(1).Infoln("Shutting down server.")

	if masterStop != nil {
		masterStop()
	}

	return nil
}

var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func SetupSignalHandler() (stopCh <-chan struct{}) {
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

func getUDSListener(ctx context.Context, udsName string) (net.Listener, error) {
	udsListenerLock.Lock()
	defer udsListenerLock.Unlock()
	oldUmask := syscall.Umask(0007)
	defer syscall.Umask(oldUmask)
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "unix", udsName)
	if err != nil {
		return nil, fmt.Errorf("failed to listen(unix) name %s: %v", udsName, err)
	}
	return lis, nil
}

func (p *Proxy) runMasterServer(ctx context.Context, o *options.ProxyRunOptions, server *server.ProxyServer) (StopFunc, error) {
	if o.UdsName != "" {
		return p.runUDSMasterServer(ctx, o, server)
	}
	return p.runMTLSMasterServer(ctx, o, server)
}

func (p *Proxy) runUDSMasterServer(ctx context.Context, o *options.ProxyRunOptions, s *server.ProxyServer) (StopFunc, error) {
	if o.DeleteUDSFile {
		if err := os.Remove(o.UdsName); err != nil && !os.IsNotExist(err) {
			klog.ErrorS(err, "failed to delete file", "file", o.UdsName)
		}
	}
	var stop StopFunc
	if o.Mode == "grpc" {
		grpcServer := grpc.NewServer()
		client.RegisterProxyServiceServer(grpcServer, s)
		lis, err := getUDSListener(ctx, o.UdsName)
		if err != nil {
			return nil, fmt.Errorf("failed to get uds listener: %v", err)
		}
		go grpcServer.Serve(lis)
		stop = grpcServer.GracefulStop
	} else {
		// http-connect
		server := &http.Server{
			Handler: &server.Tunnel{
				Server: s,
			},
		}
		stop = func() {
			err := server.Shutdown(ctx)
			klog.ErrorS(err, "error shutting down server")
		}
		go func() {
			udsListener, err := getUDSListener(ctx, o.UdsName)
			if err != nil {
				klog.ErrorS(err, "failed to get uds listener")
			}
			defer func() {
				err := udsListener.Close()
				klog.ErrorS(err, "failed to close uds listener")
			}()
			err = server.Serve(udsListener)
			if err != nil {
				klog.ErrorS(err, "failed to serve uds requests")
			}
		}()
	}

	return stop, nil
}

func (p *Proxy) getTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load X509 key pair %s and %s: %v", certFile, keyFile, err)
	}

	if caFile == "" {
		return &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}, nil
	}

	certPool := x509.NewCertPool()
	caCert, err := os.ReadFile(filepath.Clean(caFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read cluster CA cert %s: %v", caFile, err)
	}
	ok := certPool.AppendCertsFromPEM(caCert)
	if !ok {
		return nil, fmt.Errorf("failed to append cluster CA cert to the cert pool")
	}
	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS12,
	}

	return tlsConfig, nil
}

func (p *Proxy) runMTLSMasterServer(ctx context.Context, o *options.ProxyRunOptions, s *server.ProxyServer) (StopFunc, error) {
	var stop StopFunc

	var tlsConfig *tls.Config
	var err error
	if tlsConfig, err = p.getTLSConfig(o.ServerCaCert, o.ServerCert, o.ServerKey); err != nil {
		return nil, err
	}

	addr := fmt.Sprintf(":%d", o.ServerPort)

	if o.Mode == grpcMode {
		serverOption := grpc.Creds(credentials.NewTLS(tlsConfig))
		grpcServer := grpc.NewServer(serverOption)
		client.RegisterProxyServiceServer(grpcServer, s)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to listen on %s: %v", addr, err)
		}
		go grpcServer.Serve(lis)
		stop = grpcServer.GracefulStop
	} else {
		// http-connect with no tls
		httpServer := &http.Server{
			Addr: ":8088",
			Handler: &server.Tunnel{
				Server: s,
			},
		}
		// http-connect
		server := &http.Server{
			Addr:      addr,
			TLSConfig: tlsConfig,
			Handler: &server.Tunnel{
				Server: s,
			},
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}

		stop = func() {
			err := server.Shutdown(ctx)
			if err != nil {
				klog.ErrorS(err, "failed to shutdown server")
			}
			err = httpServer.Shutdown(ctx)
			if err != nil {
				klog.ErrorS(err, "failed to shutdown httpServer")
			}
		}
		go func() {
			err := server.ListenAndServeTLS("", "") // empty files defaults to tlsConfig
			if err != nil {
				klog.ErrorS(err, "failed to listen on master port")
			}
		}()
		go func() {
			err := httpServer.ListenAndServe()
			if err != nil {
				klog.ErrorS(err, "failed to listen on http master port")
			}
		}()
	}

	return stop, nil
}

func (p *Proxy) runAgentServer(o *options.ProxyRunOptions, server *server.ProxyServer) error {
	var tlsConfig *tls.Config
	var err error
	if tlsConfig, err = p.getTLSConfig(o.ClusterCaCert, o.ClusterCert, o.ClusterKey); err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", o.AgentPort)
	serverOptions := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: o.KeepaliveTime}),
	}
	grpcServer := grpc.NewServer(serverOptions...)
	agent.RegisterAgentServiceServer(grpcServer, server)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", addr, err)
	}
	go grpcServer.Serve(lis)

	return nil
}

func (p *Proxy) runAdminServer(o *options.ProxyRunOptions) {
	muxHandler := http.NewServeMux()
	muxHandler.Handle("/metrics", promhttp.Handler())
	if o.EnableProfiling {
		muxHandler.HandleFunc("/debug/pprof", util.RedirectTo("/debug/pprof/"))
		muxHandler.HandleFunc("/debug/pprof/", pprof.Index)
		if o.EnableContentionProfiling {
			runtime.SetBlockProfileRate(1)
		}
	}
	adminServer := &http.Server{
		Addr:           fmt.Sprintf("127.0.0.1:%d", o.AdminPort),
		Handler:        muxHandler,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		err := adminServer.ListenAndServe()
		if err != nil {
			klog.ErrorS(err, "admin server could not listen")
		}
		klog.V(1).Infoln("Admin server stopped listening")
	}()
}

func (p *Proxy) runHealthServer(o *options.ProxyRunOptions, server *server.ProxyServer) {
	livenessHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	readinessHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ready, msg := server.Readiness.Ready()
		if ready {
			w.WriteHeader(200)
			fmt.Fprintf(w, "ok")
			return
		}
		w.WriteHeader(500)
		fmt.Fprintf(w, msg)
	})

	muxHandler := http.NewServeMux()
	muxHandler.HandleFunc("/healthz", livenessHandler)
	muxHandler.HandleFunc("/ready", readinessHandler)
	healthServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", o.HealthPort),
		Handler:        muxHandler,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		err := healthServer.ListenAndServe()
		if err != nil {
			klog.ErrorS(err, "health server could not listen")
		}
		klog.V(1).Infoln("Health server stopped listening")
	}()
}
