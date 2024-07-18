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
package certificate

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/httpserver/resps"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/security/certs"
	"github.com/kubeedge/kubeedge/pkg/security/token"
)

// GetCA returns the caCertDER
func GetCA(_ *restful.Request, response *restful.Response) {
	resps.OK(response, hubconfig.Config.Ca)
}

// EdgeCoreClientCert will verify the certificate of EdgeCore or token then create EdgeCoreCert and return it
func EdgeCoreClientCert(request *restful.Request, response *restful.Response) {
	r := request.Request
	nodeName := r.Header.Get(types.HeaderNodeName)

	if cert := r.TLS.PeerCertificates; len(cert) > 0 {
		if err := verifyCert(cert[0], nodeName); err != nil {
			message := fmt.Sprintf("failed to verify the certificate for edgenode: %s, err: %v", nodeName, err)
			klog.Error(message)
			resps.ErrorMessage(response, http.StatusUnauthorized, message)
			return
		}
	} else {
		authorization := r.Header.Get(types.HeaderAuthorization)
		if code, err := verifyAuthorization(authorization); err != nil {
			klog.Error(err)
			resps.Error(response, code, err)
			return
		}
	}

	usagesStr := r.Header.Get(types.HeaderExtKeyUsages)
	reader := http.MaxBytesReader(response, r.Body, constants.MaxRespBodyLength)
	certBlock, err := signEdgeCert(reader, usagesStr)
	if err != nil {
		message := fmt.Sprintf("failed to sign certs for edgenode %s, err: %v", nodeName, err)
		klog.Error(message)
		resps.ErrorMessage(response, http.StatusInternalServerError, message)
		return
	}
	resps.OK(response, certBlock.Bytes)
}

// verifyCert verifies the edge certificate by CA certificate when edge certificates rotate.
func verifyCert(cert *x509.Certificate, nodeName string) error {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{
		Type:  certutil.CertificateBlockType,
		Bytes: hubconfig.Config.Ca,
	}))
	if !ok {
		return fmt.Errorf("failed to parse root certificate")
	}
	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("failed to verify edge certificate: %v", err)
	}
	return verifyCertSubject(cert, nodeName)
}

// verifyCertSubject ...
func verifyCertSubject(cert *x509.Certificate, nodeName string) error {
	if cert.Subject.Organization[0] == "KubeEdge" && cert.Subject.CommonName == "kubeedge.io" {
		// In order to maintain compatibility with older versions of certificates
		// this condition will be removed in KubeEdge v1.18.
		return nil
	}
	commonName := fmt.Sprintf("system:node:%s", nodeName)
	if cert.Subject.Organization[0] == "system:nodes" && cert.Subject.CommonName == commonName {
		return nil
	}
	return fmt.Errorf("request node name is not match with the certificate")
}

// verifyAuthorization verifies the token from EdgeCore CSR
func verifyAuthorization(authorization string) (int, error) {
	klog.V(4).Info("authorization token is: ", authorization)
	if authorization == "" {
		return http.StatusUnauthorized, errors.New("token validation failure, token is empty")
	}
	bearerToken := strings.Split(authorization, " ")
	if len(bearerToken) != 2 {
		return http.StatusUnauthorized, errors.New("token validation failure, token cannot be splited")
	}
	valid, err := token.Verify(bearerToken[1], hubconfig.Config.CaKey)
	if err != nil {
		return http.StatusUnauthorized, fmt.Errorf("token validation failure, err: %v", err)
	}
	if !valid {
		return http.StatusUnauthorized, errors.New("token validation failure, valid is false")
	}
	return http.StatusOK, nil
}

// signEdgeCert signs the CSR from EdgeCore
func signEdgeCert(r io.ReadCloser, usagesStr string) (*pem.Block, error) {
	klog.V(4).Infof("receive sign crt request, ExtKeyUsages: %s", usagesStr)
	var usages []x509.ExtKeyUsage
	if usagesStr == "" {
		usages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		err := json.Unmarshal([]byte(usagesStr), &usages)
		if err != nil {
			return nil, fmt.Errorf("unmarshal http header ExtKeyUsages fail, err: %v", err)
		}
	}
	payload, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("fail to read file when signing the cert, err: %v", err)
	}
	edgeCertSigningDuration := hubconfig.Config.CloudHub.EdgeCertSigningDuration * time.Hour * 24
	h := certs.GetHandler(certs.HandlerTypeX509)
	certBlock, err := h.SignCerts(certs.SignCertsOptionsWithCSR(
		payload,
		hubconfig.Config.Ca,
		hubconfig.Config.CaKey,
		usages,
		edgeCertSigningDuration,
	))
	if err != nil {
		return nil, fmt.Errorf("fail to signCerts, err: %v", err)
	}
	return certBlock, nil
}
