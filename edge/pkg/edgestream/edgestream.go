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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	"github.com/kubeedge/kubeedge/edge/pkg/edgestream/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/stream"
	"github.com/kubeedge/kubeedge/pkg/util"
)

type edgestream struct {
	enable           bool
	hostnameOverride string
	nodeIP           string
}

var _ core.Module = (*edgestream)(nil)

func newEdgeStream(enable bool, hostnameOverride, nodeIP string) (*edgestream, error) {
	var err error
	if nodeIP == "" {
		nodeIP, err = util.GetLocalIP(util.GetHostname())
		if err != nil {
			klog.Errorf("Failed to get Local IP address: %v", err)
			return nil, err
		}
		klog.Infof("Get node local IP address successfully: %s", nodeIP)
	}

	return &edgestream{
		enable:           enable,
		hostnameOverride: hostnameOverride,
		nodeIP:           nodeIP,
	}, nil
}

// Register register edgestream
func Register(s *v1alpha2.EdgeStream, hostnameOverride, nodeIP string) {
	config.InitConfigure(s)
	edgeStream, err := newEdgeStream(s.Enable, hostnameOverride, nodeIP)
	if err != nil {
		klog.Errorf("init new edged error, %v", err)
		os.Exit(1)
	}
	core.Register(edgeStream)
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
	serverURL := url.URL{
		Scheme: "wss",
		Host:   config.Config.TunnelServer,
		Path:   "/v1/kubeedge/connect",
	}
	// TODO: Will improve in the future
	if ok := <-edgehub.GetCertSyncChannel()[e.Name()]; !ok {
		panic(fmt.Errorf("Failed to find cert key pair"))
	}

	cert, err := tls.LoadX509KeyPair(config.Config.TLSTunnelCertFile, config.Config.TLSTunnelPrivateKeyFile)
	if err != nil {
		panic(fmt.Errorf("Failed to load x509 key pair: %v", err))
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()

	for {
		select {
		case <-beehiveContext.Done():
			return
		case <-ticker.C:
			err := e.TLSClientConnect(serverURL, tlsConfig)
			if err != nil {
				klog.Errorf("TLSClientConnect error %v", err)
			}
		}
	}
}

func (e *edgestream) TLSClientConnect(url url.URL, tlsConfig *tls.Config) error {
	klog.Info("Start a new tunnel stream connection ...")

	dial := websocket.Dialer{
		TLSClientConfig:  tlsConfig,
		HandshakeTimeout: time.Duration(config.Config.HandshakeTimeout) * time.Second,
	}
	header := http.Header{}
	header.Add(stream.SessionKeyHostNameOverride, e.hostnameOverride)
	header.Add(stream.SessionKeyInternalIP, e.nodeIP)

	con, _, err := dial.Dial(url.String(), header)
	if err != nil {
		klog.Errorf("dial %v error %v", url.String(), err)
		return err
	}
	session := NewTunnelSession(con)
	return session.Serve()
}
