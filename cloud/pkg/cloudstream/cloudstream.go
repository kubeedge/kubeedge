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

package cloudstream

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"k8s.io/klog/v2"
	anpserver "sigs.k8s.io/apiserver-network-proxy/pkg/server"
	"sigs.k8s.io/apiserver-network-proxy/proto/agent"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/cert"
)

var udsListenerLock sync.Mutex

type cloudStream struct {
	enable bool
}

func newCloudStream(enable bool) *cloudStream {
	return &cloudStream{
		enable: enable,
	}
}

func Register(c *v1alpha1.CloudStream) {
	config.InitConfigure(c)
	core.Register(newCloudStream(c.Enable))
}

func (c *cloudStream) Name() string {
	return modules.CloudStreamModuleName
}

func (c *cloudStream) Group() string {
	return modules.CloudStreamGroupName
}

func (c *cloudStream) Enable() bool {
	return c.enable
}

func (c *cloudStream) Start() {
	// wait certs generate
	<-cloudhub.DoneTLSTunnelCerts

	klog.Infoln("starting proxy server")
	proxyServer := anpserver.NewProxyServer(uuid.New().String(), config.Config.ServerAccount, &anpserver.AgentTokenAuthenticationOptions{})
	err := runProxyServer(&anpserver.Tunnel{Server: proxyServer}, config.Config.UDSFile)
	if err != nil {
		klog.Errorf("failed to run the proxy server: %v", err)
	}

	klog.Infoln("starting mater server")
	var tlsConfig *tls.Config
	if tlsConfig, err = cert.GetTLSConfig(config.Config.TLSTunnelCAFile, config.Config.TLSTunnelCertFile, config.Config.TLSTunnelPrivateKeyFile); err != nil {
		klog.Errorf("failed to get tls config: %v", err)
	}
	err = runMasterServer(WrapHandler(&WarpHandler{UDSSockFile: config.Config.UDSFile, TLSConfig: tlsConfig}), tlsConfig)
	if err != nil {
		klog.Errorf("failed to run the master server: %v", err)
	}

	klog.Infoln("starting agent server for edge connections")
	err = c.runAgentServer(proxyServer)
	if err != nil {
		klog.Errorf("failed to run the agent server: %v", err)
	}
}

// runProxyServer starts a proxy server that redirects requests received from
// apiserver to edge
func runProxyServer(handler http.Handler, udsSockFile string) error {
	// request will be sent from request interceptor on the same host,
	// so we use UDS protocol to avoide sending request through kernel
	// network stack.
	go func() {
		server := &http.Server{
			Handler: handler,
		}
		udsListenerLock.Lock()
		defer udsListenerLock.Unlock()
		unixListener, err := net.Listen("unix", udsSockFile)
		if err != nil {
			klog.Errorf("failed to request through uds: %s", err)
		}
		defer unixListener.Close()
		if err := server.Serve(unixListener); err != nil {
			klog.Errorf("failed to request through uds: %s", err)
		}
	}()

	return nil
}

// runMasterServer runs an https server to handle requests from apiserver
func runMasterServer(handler http.Handler, tlsConfig *tls.Config) error {
	go func() {
		server := http.Server{
			Addr:         fmt.Sprintf(":%d", config.Config.TunnelPort),
			Handler:      handler,
			TLSConfig:    tlsConfig,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}
		// set empty to use TLSConfig
		if err := server.ListenAndServeTLS("", ""); err != nil {
			klog.Errorf("failed to serve https request from master: %v", err)
		}
	}()

	return nil
}

func (c *cloudStream) runAgentServer(server *anpserver.ProxyServer) error {
	var tlsConfig *tls.Config
	var err error
	if tlsConfig, err = cert.GetTLSConfigFromBytes(hubconfig.Config.Ca, hubconfig.Config.Cert, hubconfig.Config.Key); err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", config.Config.AgentPort)
	serverOptions := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: constants.GrpcKeepAliveTimeSec * time.Second}),
	}
	grpcServer := grpc.NewServer(serverOptions...)
	agent.RegisterAgentServiceServer(grpcServer, server)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", addr, err)
	}
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			klog.Error("failed to run agent server")
		}
	}()

	return nil
}
