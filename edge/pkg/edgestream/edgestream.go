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
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	"github.com/kubeedge/kubeedge/edge/pkg/edgestream/config"
	"github.com/kubeedge/kubeedge/pkg/features"
	"github.com/kubeedge/kubeedge/pkg/stream"
	"github.com/kubeedge/kubeedge/pkg/util"
)

type edgestream struct {
	enable           bool
	hostnameOverride string
	nodeIP           string
}

var _ core.Module = (*edgestream)(nil)

func newEdgeStream(enable bool, hostnameOverride, nodeIP string) *edgestream {
	return &edgestream{
		enable:           enable,
		hostnameOverride: hostnameOverride,
		nodeIP:           nodeIP,
	}
}

// Register register edgestream
func Register(s *v1alpha2.EdgeStream, hostnameOverride, nodeIP string) {
	config.InitConfigure(s)
	core.Register(newEdgeStream(s.Enable, hostnameOverride, nodeIP))
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

func (e *edgestream) RestartPolicy() *core.ModuleRestartPolicy {
	if !features.DefaultFeatureGate.Enabled(features.ModuleRestart) {
		return nil
	}
	return &core.ModuleRestartPolicy{
		RestartType:            core.RestartTypeOnFailure,
		IntervalTimeGrowthRate: 2.0,
	}
}

func (e *edgestream) Start() {
	serverURL := url.URL{
		Scheme: "wss",
		Host:   config.Config.TunnelServer,
		Path:   "/v1/kubeedge/connect",
	}
	// TODO: Will improve in the future
	if ok := <-edgehub.GetCertSyncChannel()[e.Name()]; !ok {
		klog.Exitf("Failed to find cert key pair")
	}

	reloader, err := newCertReloader(config.Config.TLSTunnelCertFile, config.Config.TLSTunnelPrivateKeyFile)
	if err != nil {
		klog.Exitf("Failed to load x509 key pair: %v", err)
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify:   true,
		GetClientCertificate: reloader.GetClientCertificate,
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

// certReloader reads the tunnel client certificate from disk on every TLS
// handshake, so a certificate rotated by EdgeHub is used on the next reconnect
// instead of the one loaded at startup, which CloudStream rejects once expired.
type certReloader struct {
	certFile string
	keyFile  string

	mu   sync.Mutex
	cert *tls.Certificate
}

func newCertReloader(certFile, keyFile string) (*certReloader, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &certReloader{certFile: certFile, keyFile: keyFile, cert: &cert}, nil
}

// GetClientCertificate implements tls.Config.GetClientCertificate. If the key
// pair cannot be loaded, e.g. while rotation has written the new certificate
// but not yet the new key, the last successfully loaded pair is returned.
func (r *certReloader) GetClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)

	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		klog.Warningf("Failed to reload x509 key pair, using the previously loaded one: %v", err)
		return r.cert, nil
	}
	r.cert = &cert
	return r.cert, nil
}

func (e *edgestream) TLSClientConnect(url url.URL, tlsConfig *tls.Config) error {
	klog.Info("Start a new tunnel stream connection ...")

	// If the node IP address is not specified in the configuration file,
	// the node IP address is reacquired each time the tunnel stream is reconnected
	var nodeIP string
	if e.nodeIP == "" {
		ip, err := util.GetLocalIP(util.GetHostname())
		if err != nil {
			return fmt.Errorf("failed to get Local IP address: %v", err)
		}
		klog.Infof("get node local IP address successfully: %s", ip)
		nodeIP = ip
	} else {
		nodeIP = e.nodeIP
	}

	dial := websocket.Dialer{
		TLSClientConfig:  tlsConfig,
		HandshakeTimeout: time.Duration(config.Config.HandshakeTimeout) * time.Second,
	}
	header := http.Header{}
	header.Add(stream.SessionKeyHostNameOverride, e.hostnameOverride)
	header.Add(stream.SessionKeyInternalIP, nodeIP)

	con, _, err := dial.Dial(url.String(), header)
	if err != nil {
		klog.Errorf("dial %v error %v", url.String(), err)
		return err
	}
	session := NewTunnelSession(con)
	return session.Serve()
}
