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

package edgestream

import (
	"crypto/tls"
	"time"

	"github.com/google/uuid"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	hubconfig "github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgestream/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/cert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/klog/v2"
	"sigs.k8s.io/apiserver-network-proxy/pkg/agent"
)

type edgestream struct {
	enable bool
}

func newEdgeStream(enable bool) *edgestream {
	return &edgestream{
		enable: enable,
	}
}

// Register register edgestream
func Register(e *v1alpha1.EdgeStream) {
	config.InitConfigure(e)
	core.Register(newEdgeStream(e.Enable))
}

func (e *edgestream) Name() string {
	return modules.EdgeStreamModuleName
}

func (e *edgestream) Group() string {
	return modules.StreamGroup
}

func (e *edgestream) Enable() bool {
	return e.enable
}

func (e *edgestream) Start() {
	<-edgehub.HasTLSTunnelCerts
	stopCh := make(chan struct{})
	if err := e.runProxyConnection(stopCh); err != nil {
		klog.Errorf("failed to run edgestream, err %v", err)
	}
}

func (e *edgestream) runProxyConnection(stopCh <-chan struct{}) error {
	var tlsConfig *tls.Config
	var err error
	if tlsConfig, err = cert.GetTLSConfig(hubconfig.Config.TLSCAFile, hubconfig.Config.TLSCertFile, hubconfig.Config.TLSPrivateKeyFile); err != nil {
		return err
	}

	dialOption := grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	cc := &agent.ClientSetConfig{
		Address:                 config.Config.AgentServer,
		AgentID:                 uuid.New().String(),
		SyncInterval:            time.Duration(config.Config.SyncInterval) * time.Second,
		ProbeInterval:           time.Duration(config.Config.ProbeInterval) * time.Second,
		DialOptions:             []grpc.DialOption{dialOption},
		ServiceAccountTokenPath: "",
	}
	cs := cc.NewAgentClientSet(stopCh)
	cs.Serve()

	return nil
}
