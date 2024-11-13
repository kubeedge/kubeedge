/*
Copyright 2024 The KubeEdge Authors.

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

package authorization

import (
	"context"
	"crypto/tls"
	stdx509 "crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"

	"k8s.io/apiserver/pkg/authentication/request/x509"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/endpoints/request"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	cloudhubmodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn"
)

type cloudhubAuthorizer struct {
	enabled bool
	debug   bool
	authz   authorizer.Authorizer
}

func (r *cloudhubAuthorizer) AdmitMessage(message beehivemodel.Message, hubInfo cloudhubmodel.HubInfo) error {
	if !r.enabled {
		return nil
	}

	err := r.admitMessage(message, hubInfo)
	if err == nil {
		return nil
	}

	klog.Error(err.Error())
	if r.debug {
		return nil
	}
	return err
}

func (r *cloudhubAuthorizer) AuthenticateConnection(connection conn.Connection) error {
	if !r.enabled {
		return nil
	}

	err := r.authenticateConnection(connection)
	if err == nil {
		return nil
	}

	klog.Error(err.Error())
	if r.debug {
		return nil
	}
	return err
}

// admitMessage determines whether the message should be admitted.
func (r *cloudhubAuthorizer) admitMessage(message beehivemodel.Message, hubInfo cloudhubmodel.HubInfo) error {
	klog.V(4).Infof("message: %s: authorization start", message.Header.ID)

	attrs, err := getAuthorizerAttributes(message.Router, hubInfo)
	if err != nil {
		return fmt.Errorf("node %q transfer message failed: %v", hubInfo.NodeID, err)
	}

	ctx := request.WithUser(context.TODO(), attrs.GetUser())
	authorized, reason, err := r.authz.Authorize(ctx, attrs)
	if err != nil {
		return fmt.Errorf("node %q authz failed: %v", hubInfo.NodeID, err)
	}

	if authorized != authorizer.DecisionAllow {
		return fmt.Errorf("node %q deny: %s", hubInfo.NodeID, reason)
	}

	klog.V(4).Infof("message: %s: authorization succeeded", message.Header.ID)
	return nil
}

// authenticateConnection authenticates the new connection by certificates
func (r *cloudhubAuthorizer) authenticateConnection(connection conn.Connection) error {
	peerCerts := connection.ConnectionState().PeerCertificates
	nodeID := connection.ConnectionState().Headers.Get("node_id")

	klog.V(4).Infof("node %q: authentication start", nodeID)
	switch len(peerCerts) {
	case 0:
		return fmt.Errorf("node %q: no client certificate provided", nodeID)
	case 1:
	default:
		return fmt.Errorf("node %q: immediate certificates are not supported", nodeID)
	}

	options := x509.DefaultVerifyOptions()
	// ca cloud be available util CloudHub starts
	options.Roots = stdx509.NewCertPool()
	options.Roots.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: hubconfig.Config.Ca}))

	authenticator := x509.New(options, x509.CommonNameUserConversion)
	resp, ok, err := authenticator.AuthenticateRequest(&http.Request{TLS: &tls.ConnectionState{PeerCertificates: peerCerts}})
	if err != nil || !ok {
		return fmt.Errorf("node %q: unable to verify peer connection by client certificates: %v", nodeID, err)
	}

	if resp.User.GetName() != constants.NodesUserPrefix+nodeID {
		return fmt.Errorf("node %q: common name of peer certificate didn't match node ID", nodeID)
	}

	klog.V(4).Infof("node %q: authentication succeeded", nodeID)
	return nil
}
