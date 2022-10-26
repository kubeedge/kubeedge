/*
Copyright 2022 The KubeEdge Authors.

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

package filter

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
	kefeatures "github.com/kubeedge/kubeedge/pkg/features"
)

func verifyNodeCert(peerCert []*x509.Certificate, nodeName string) error {
	var err error
	if len(peerCert) == 0 {
		err = fmt.Errorf("tls certificates can't be empty")
		klog.Errorf(err.Error())
		return err
	}
	if len(peerCert[0].Subject.CommonName) == 0 {
		err = fmt.Errorf("invalid length of tls certificates common name")
		klog.Errorf(err.Error())
		return err
	}
	cn := peerCert[0].Subject.CommonName
	if !strings.HasPrefix(cn, constants.NodeUserNamePrefix) {
		err = fmt.Errorf("invalid tls certificates common name prefix: %v", cn)
		klog.Errorf(err.Error())
		return err
	}
	certNodeName := strings.TrimPrefix(cn, constants.NodeUserNamePrefix)
	if certNodeName != nodeName {
		err = fmt.Errorf("node %q is not allowed to modify node %q", certNodeName, nodeName)
		klog.Errorf(err.Error())
		return err
	}
	return nil
}

func QuicFilter(certChain interface{}, header *http.Header) error {
	if !kefeatures.DefaultFeatureGate.Enabled(kefeatures.NodeAttestation) {
		return nil
	}
	cert, ok := certChain.([]*x509.Certificate)
	if !ok {
		klog.Errorf("invalid connection type: %T", QuicFilter)
		return fmt.Errorf("invalid connection type: %T", QuicFilter)
	}
	err := verifyNodeCert(cert, header.Get("node_id"))
	if err != nil {
		return err
	}
	return nil
}

func WsFilter(w http.ResponseWriter, req *http.Request) error {
	if !kefeatures.DefaultFeatureGate.Enabled(kefeatures.NodeAttestation) {
		return nil
	}
	if req.TLS == nil {
		klog.Errorf("invalid tls certificates")
		return fmt.Errorf("invalid tls certificates")
	}
	err := verifyNodeCert(req.TLS.PeerCertificates, req.Header.Get("node_id"))
	if err != nil {
		return err
	}
	return nil
}
