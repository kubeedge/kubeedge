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
	"time"

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
		klog.V(5).Infof("Authorization disabled, admitting message %s from node %s", message.Header.ID, hubInfo.NodeID)
		return nil
	}

	startTime := time.Now()
	err := r.admitMessage(message, hubInfo)
	duration := time.Since(startTime)

	if err == nil {
		klog.V(4).Infof("Message admission successful: id=%s, node=%s, resource=%s, operation=%s, duration=%v", 
			message.Header.ID, hubInfo.NodeID, message.Router.Resource, message.Router.Operation, duration)
		return nil
	}

	// Log detailed audit failure
	klog.Warningf("Message admission failed: id=%s, node=%s, resource=%s, operation=%s, error=%v, duration=%v",
		message.Header.ID, hubInfo.NodeID, message.Router.Resource, message.Router.Operation, err, duration)
	
	if r.debug {
		klog.V(3).Infof("Debug mode enabled, admitting message despite authorization failure: %s", message.Header.ID)
		return nil
	}
	return err
}

func (r *cloudhubAuthorizer) AuthenticateConnection(connection conn.Connection) error {
	if !r.enabled {
		klog.V(5).Infof("Authentication disabled, allowing connection")
		return nil
	}

	startTime := time.Now()
	err := r.authenticateConnection(connection)
	duration := time.Since(startTime)

	nodeID := connection.ConnectionState().Headers.Get("node_id")
	peerCerts := connection.ConnectionState().PeerCertificates

	if err == nil {
		// Log successful authentication with certificate details
		var certInfo string
		if len(peerCerts) > 0 {
			cert := peerCerts[0]
			certInfo = fmt.Sprintf(", cert_subject=%s, cert_issuer=%s, cert_expiry=%s", 
				cert.Subject.CommonName, cert.Issuer.CommonName, cert.NotAfter.Format(time.RFC3339))
		}
		klog.V(4).Infof("Connection authentication successful: node=%s, duration=%v%s", 
			nodeID, duration, certInfo)
		return nil
	}

	// Log authentication failure with details
	klog.Warningf("Connection authentication failed: node=%s, error=%v, duration=%v", 
		nodeID, err, duration)
	
	if r.debug {
		klog.V(3).Infof("Debug mode enabled, allowing connection despite authentication failure for node: %s", nodeID)
		return nil
	}
	return err
}

// admitMessage determines whether the message should be admitted.
func (r *cloudhubAuthorizer) admitMessage(message beehivemodel.Message, hubInfo cloudhubmodel.HubInfo) error {
	klog.V(4).Infof("Authorization starting: message_id=%s, node=%s, resource=%s, operation=%s", 
		message.Header.ID, hubInfo.NodeID, message.Router.Resource, message.Router.Operation)

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
		return fmt.Errorf("node %q authorization denied: %s", hubInfo.NodeID, reason)
	}

	klog.V(4).Infof("Authorization completed successfully: message_id=%s, node=%s", 
		message.Header.ID, hubInfo.NodeID)
	return nil
}

// authenticateConnection authenticates the new connection by certificates
func (r *cloudhubAuthorizer) authenticateConnection(connection conn.Connection) error {
	peerCerts := connection.ConnectionState().PeerCertificates
	nodeID := connection.ConnectionState().Headers.Get("node_id")

	klog.V(4).Infof("Authentication starting: node=%s, peer_cert_count=%d", nodeID, len(peerCerts))
	
	switch len(peerCerts) {
	case 0:
		return fmt.Errorf("node %q: no client certificate provided", nodeID)
	case 1:
		// Log certificate details for audit purposes
		cert := peerCerts[0]
		klog.V(5).Infof("Peer certificate details: node=%s, subject=%s, issuer=%s, expiry=%s", 
			nodeID, cert.Subject.CommonName, cert.Issuer.CommonName, cert.NotAfter.Format(time.RFC3339))
		
		// Check certificate expiry and log warning if expiring soon
		if time.Until(cert.NotAfter) < 30*24*time.Hour { // 30 days
			klog.Warningf("Certificate for node %s is expiring soon: %s", nodeID, cert.NotAfter.Format(time.RFC3339))
		}
	default:
		return fmt.Errorf("node %q: intermediate certificates are not supported, received %d certificates", nodeID, len(peerCerts))
	}

	options := x509.DefaultVerifyOptions()
	// ca could be available until CloudHub starts
	options.Roots = stdx509.NewCertPool()
	options.Roots.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: hubconfig.Config.Ca}))

	authenticator := x509.New(options, x509.CommonNameUserConversion)
	resp, ok, err := authenticator.AuthenticateRequest(&http.Request{TLS: &tls.ConnectionState{PeerCertificates: peerCerts}})
	if err != nil || !ok {
		return fmt.Errorf("node %q: unable to verify peer connection by client certificates: %v", nodeID, err)
	}

	if resp.User.GetName() != constants.NodesUserPrefix+nodeID {
		return fmt.Errorf("node %q: common name of peer certificate didn't match node ID, expected %s, got %s", 
			nodeID, constants.NodesUserPrefix+nodeID, resp.User.GetName())
	}

	klog.V(4).Infof("Authentication completed successfully: node=%s", nodeID)
	return nil
}