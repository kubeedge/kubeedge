/*
Copyright 2022 The Kubernetes Authors.

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

package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	runpprof "runtime/pprof"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"k8s.io/klog/v2"

	"sigs.k8s.io/apiserver-network-proxy/cmd/agent/app/options"
	"sigs.k8s.io/apiserver-network-proxy/pkg/util"
)

const ReadHeaderTimeout = 60 * time.Second

func NewAgentCommand(a *Agent, o *options.GrpcProxyAgentOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "agent",
		Long: `A gRPC agent, Connects to the proxy and then allows traffic to be forwarded to it.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.run(o)
		},
	}

	return cmd
}

type Agent struct {
}

func (a *Agent) run(o *options.GrpcProxyAgentOptions) error {
	o.Print()
	if err := o.Validate(); err != nil {
		return fmt.Errorf("failed to validate agent options with %v", err)
	}

	stopCh := make(chan struct{})
	if err := a.runProxyConnection(o, stopCh); err != nil {
		return fmt.Errorf("failed to run proxy connection with %v", err)
	}

	if err := a.runHealthServer(o); err != nil {
		return fmt.Errorf("failed to run health server with %v", err)
	}

	if err := a.runAdminServer(o); err != nil {
		return fmt.Errorf("failed to run admin server with %v", err)
	}

	<-stopCh

	return nil
}

func (a *Agent) runProxyConnection(o *options.GrpcProxyAgentOptions, stopCh <-chan struct{}) error {
	var tlsConfig *tls.Config
	var err error
	if tlsConfig, err = util.GetClientTLSConfig(o.CaCert, o.AgentCert, o.AgentKey, o.ProxyServerHost, o.AlpnProtos); err != nil {
		return err
	}
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                o.KeepaliveTime,
			PermitWithoutStream: true,
		}),
	}
	cc := o.ClientSetConfig(dialOptions...)
	cs := cc.NewAgentClientSet(stopCh)
	cs.Serve()

	return nil
}

func (a *Agent) runHealthServer(o *options.GrpcProxyAgentOptions) error {
	livenessHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	readinessHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})

	muxHandler := http.NewServeMux()
	muxHandler.Handle("/metrics", promhttp.Handler())
	muxHandler.HandleFunc("/healthz", livenessHandler)
	// "/ready" is deprecated but being maintained for backward compatibility
	muxHandler.HandleFunc("/ready", readinessHandler)
	muxHandler.HandleFunc("/readyz", readinessHandler)
	healthServer := &http.Server{
		Addr:              net.JoinHostPort(o.HealthServerHost, strconv.Itoa(o.HealthServerPort)),
		Handler:           muxHandler,
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: ReadHeaderTimeout,
	}

	labels := runpprof.Labels(
		"core", "healthListener",
		"port", strconv.Itoa(o.HealthServerPort),
	)
	go runpprof.Do(context.Background(), labels, func(context.Context) { a.serveHealth(healthServer) })

	return nil
}

func (a *Agent) serveHealth(healthServer *http.Server) {
	err := healthServer.ListenAndServe()
	if err != nil {
		klog.ErrorS(err, "health server could not listen")
	}
	klog.V(0).Infoln("Health server stopped listening")
}

func (a *Agent) runAdminServer(o *options.GrpcProxyAgentOptions) error {
	muxHandler := http.NewServeMux()
	muxHandler.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		// The port number may be omitted if the admin server is running on port
		// 80, the default port for HTTP
		if err != nil {
			host = r.Host
		}
		http.Redirect(w, r, fmt.Sprintf("%s:%d%s", host, o.HealthServerPort, r.URL.Path), http.StatusMovedPermanently)
	}))
	if o.EnableProfiling {
		muxHandler.HandleFunc("/debug/pprof", util.RedirectTo("/debug/pprof/"))
		muxHandler.HandleFunc("/debug/pprof/", pprof.Index)
		muxHandler.HandleFunc("/debug/pprof/profile", pprof.Profile)
		muxHandler.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		muxHandler.HandleFunc("/debug/pprof/trace", pprof.Trace)
		if o.EnableContentionProfiling {
			runtime.SetBlockProfileRate(1)
		}
	}

	adminServer := &http.Server{
		Addr:              net.JoinHostPort(o.AdminBindAddress, strconv.Itoa(o.AdminServerPort)),
		Handler:           muxHandler,
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: ReadHeaderTimeout,
	}

	labels := runpprof.Labels(
		"core", "adminListener",
		"port", strconv.Itoa(o.AdminServerPort),
	)
	go runpprof.Do(context.Background(), labels, func(context.Context) { a.serveAdmin(adminServer) })

	return nil
}

func (a *Agent) serveAdmin(adminServer *http.Server) {
	err := adminServer.ListenAndServe()
	if err != nil {
		klog.ErrorS(err, "admin server could not listen")
	}
	klog.V(0).Infoln("Admin server stopped listening")
}
