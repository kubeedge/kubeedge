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

package httpserver

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/golang-jwt/jwt"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/common/constants"
)

// StartHTTPServer starts the http service
func StartHTTPServer() {
	serverContainer := restful.NewContainer()
	ws := new(restful.WebService)
	ws.Path("/")
	ws.Route(ws.GET(constants.DefaultCertURL).To(edgeCoreClientCert))
	ws.Route(ws.GET(constants.DefaultCAURL).To(getCA))
	ws.Route(ws.POST(constants.DefaultNodeUpgradeURL).To(upgradeEdge))
	serverContainer.Add(ws)

	addr := fmt.Sprintf("%s:%d", hubconfig.Config.HTTPS.Address, hubconfig.Config.HTTPS.Port)

	cert, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: hubconfig.Config.Cert}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: hubconfig.Config.Key}))

	if err != nil {
		klog.Exit(err)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: serverContainer,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequestClientCert,
		},
	}
	klog.Exit(server.ListenAndServeTLS("", ""))
}

// getCA returns the caCertDER
func getCA(request *restful.Request, response *restful.Response) {
	caCertDER := hubconfig.Config.Ca
	if _, err := response.Write(caCertDER); err != nil {
		klog.Errorf("failed to write caCertDER, err: %v", err)
	}
}

// EncodeCertPEM returns PEM-encoded certificate data
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  certutil.CertificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// edgeCoreClientCert will verify the certificate of EdgeCore or token then create EdgeCoreCert and return it
func edgeCoreClientCert(request *restful.Request, response *restful.Response) {
	if cert := request.Request.TLS.PeerCertificates; len(cert) > 0 {
		if err := verifyCert(cert[0]); err != nil {
			klog.Errorf("failed to sign the certificate for edgenode: %s, failed to verify the certificate", request.Request.Header.Get(constants.NodeName))
			response.WriteHeader(http.StatusUnauthorized)
			if _, err := response.Write([]byte(err.Error())); err != nil {
				klog.Errorf("failed to write response, err: %v", err)
			}
		} else {
			signEdgeCert(response, request.Request)
		}
		return
	}
	if verifyAuthorization(response, request.Request) {
		signEdgeCert(response, request.Request)
	} else {
		klog.Errorf("failed to sign the certificate for edgenode: %s, invalid token", request.Request.Header.Get(constants.NodeName))
	}
}

// verifyCert verifies the edge certificate by CA certificate when edge certificates rotate.
func verifyCert(cert *x509.Certificate) error {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: hubconfig.Config.Ca}))
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
	return nil
}

// verifyAuthorization verifies the token from EdgeCore CSR
func verifyAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authorizationHeader := r.Header.Get("authorization")
	if authorizationHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte("Invalid authorization token")); err != nil {
			klog.Errorf("failed to write http response, err: %v", err)
		}
		return false
	}
	bearerToken := strings.Split(authorizationHeader, " ")
	if len(bearerToken) != 2 {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte("Invalid authorization token")); err != nil {
			klog.Errorf("failed to write http response, err: %v", err)
		}
		return false
	}
	token, err := jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("there was an error")
		}
		caKey := hubconfig.Config.CaKey
		return caKey, nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			w.WriteHeader(http.StatusUnauthorized)
			if _, err := w.Write([]byte("Invalid authorization token")); err != nil {
				klog.Errorf("Write body error %v", err)
			}
			return false
		}
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("Invalid authorization token")); err != nil {
			klog.Errorf("Write body error %v", err)
		}

		return false
	}
	if !token.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte("Invalid authorization token")); err != nil {
			klog.Errorf("Write body error %v", err)
		}
		return false
	}
	return true
}

// signEdgeCert signs the CSR from EdgeCore
func signEdgeCert(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, constants.MaxRespBodyLength)
	csrContent, err := io.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("fail to read file when signing the cert for edgenode:%s! error:%v", r.Header.Get(constants.NodeName), err)
		return
	}
	csr, err := x509.ParseCertificateRequest(csrContent)
	if err != nil {
		klog.Errorf("fail to ParseCertificateRequest of edgenode: %s! error:%v", r.Header.Get(constants.NodeName), err)
		return
	}
	usagesStr := r.Header.Get("ExtKeyUsages")
	var usages []x509.ExtKeyUsage
	if usagesStr == "" {
		usages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		err := json.Unmarshal([]byte(usagesStr), &usages)
		if err != nil {
			klog.Errorf("unmarshal http header ExtKeyUsages fail, err: %v", err)
			return
		}
	}
	klog.V(4).Infof("receive sign crt request, ExtKeyUsages: %v", usages)
	clientCertDER, err := signCerts(csr.Subject, csr.PublicKey, usages)
	if err != nil {
		klog.Errorf("fail to signCerts for edgenode:%s! error:%v", r.Header.Get(constants.NodeName), err)
		return
	}

	if _, err := w.Write(clientCertDER); err != nil {
		klog.Errorf("write error %v", err)
	}
}

// signCerts will create a certificate for EdgeCore
func signCerts(subInfo pkix.Name, pbKey crypto.PublicKey, usages []x509.ExtKeyUsage) ([]byte, error) {
	cfgs := &certutil.Config{
		CommonName:   subInfo.CommonName,
		Organization: subInfo.Organization,
		Usages:       usages,
	}
	clientKey := pbKey

	ca := hubconfig.Config.Ca
	caCert, err := x509.ParseCertificate(ca)
	if err != nil {
		return nil, fmt.Errorf("unable to ParseCertificate: %v", err)
	}

	caKeyDER := hubconfig.Config.CaKey
	caKey, err := x509.ParseECPrivateKey(caKeyDER)
	if err != nil {
		return nil, fmt.Errorf("unable to ParseECPrivateKey: %v", err)
	}

	edgeCertSigningDuration := hubconfig.Config.CloudHub.EdgeCertSigningDuration
	certDER, err := NewCertFromCa(cfgs, caCert, clientKey, caKey, edgeCertSigningDuration) //crypto.Signer(caKey)
	if err != nil {
		return nil, fmt.Errorf("unable to NewCertFromCa: %v", err)
	}

	return certDER, err
}

// PrepareAllCerts check whether the certificates exist in the local directory,
// and then check whether certificates exist in the secret, generate if they don't exist
func PrepareAllCerts() error {
	// Check whether the ca exists in the local directory
	if hubconfig.Config.Ca == nil && hubconfig.Config.CaKey == nil {
		var caDER, caKeyDER []byte
		klog.Info("Ca and CaKey don't exist in local directory, and will read from the secret")
		// Check whether the ca exists in the secret
		caSecret, err := GetSecret(CaSecretName, constants.SystemNamespace)
		if err != nil {
			klog.Info("Ca and CaKey don't exist in the secret, and will be created by CloudCore")
			var caKey crypto.Signer
			caDER, caKey, err = NewCertificateAuthorityDer()
			if err != nil {
				klog.Errorf("failed to create Certificate Authority, error: %v", err)
				return err
			}

			caKeyDER, err = x509.MarshalECPrivateKey(caKey.(*ecdsa.PrivateKey))
			if err != nil {
				klog.Errorf("failed to convert an EC private key to SEC 1, ASN.1 DER form, error: %v", err)
				return err
			}

			err = CreateCaSecret(caDER, caKeyDER)
			if err != nil {
				klog.Errorf("failed to create ca to secrets, error: %v", err)
				return err
			}
		} else {
			caDER = caSecret.Data[CaDataName]
			caKeyDER = caSecret.Data[CaKeyDataName]
		}
		UpdateConfig(caDER, caKeyDER, nil, nil)
	} else {
		// HubConfig has been initialized
		if err := CreateCaSecret(hubconfig.Config.Ca, hubconfig.Config.CaKey); err != nil {
			klog.Errorf("failed to save ca and key to the secret, error: %v", err)
			return err
		}
	}

	// Check whether the CloudCore certificates exist in the local directory
	if hubconfig.Config.Key == nil && hubconfig.Config.Cert == nil {
		klog.Infof("CloudCoreCert and key don't exist in local directory, and will read from the secret")
		// Check whether the CloudCore certificates exist in the secret
		var certDER, keyDER []byte
		cloudSecret, err := GetSecret(CloudCoreSecretName, constants.SystemNamespace)
		if err != nil {
			klog.Info("CloudCoreCert and key don't exist in the secret, and will be signed by CA")
			certDER, keyDER, err = SignCerts()
			if err != nil {
				klog.Errorf("failed to sign a certificate, error: %v", err)
				return err
			}

			err = CreateCloudCoreSecret(certDER, keyDER)
			if err != nil {
				klog.Errorf("failed to save CloudCore cert and key to secret, error: %v", err)
				return err
			}
		} else {
			certDER = cloudSecret.Data[CloudCoreCertName]
			keyDER = cloudSecret.Data[CloudCoreKeyDataName]
		}
		UpdateConfig(nil, nil, certDER, keyDER)
	} else {
		// HubConfig has been initialized
		if err := CreateCloudCoreSecret(hubconfig.Config.Cert, hubconfig.Config.Key); err != nil {
			klog.Errorf("failed to save CloudCore cert to secret, error: %v", err)
			return err
		}
	}
	return nil
}
