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
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/common/constants"
)

const (
	certificateBlockType = "CERTIFICATE"
)

// StartHTTPServer starts the http service
func StartHTTPServer() {
	router := mux.NewRouter()
	router.HandleFunc(constants.DefaultCertURL, edgeCoreClientCert).Methods("GET")
	router.HandleFunc(constants.DefaultCAURL, getCA).Methods("GET")
	router.HandleFunc(constants.DefaultCloudCoreReadyCheckURL, electionHandler).Methods("GET")

	addr := fmt.Sprintf("%s:%d", hubconfig.Config.HTTPS.Address, hubconfig.Config.HTTPS.Port)

	cert, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: hubconfig.Config.Cert}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: hubconfig.Config.Key}))

	if err != nil {
		klog.Fatal(err)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: router,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequestClientCert,
		},
	}
	klog.Fatal(server.ListenAndServeTLS("", ""))
}

// getCA returns the caCertDER
func getCA(w http.ResponseWriter, r *http.Request) {
	caCertDER := hubconfig.Config.Ca
	if _, err := w.Write(caCertDER); err != nil {
		klog.Errorf("failed to write caCertDER, err: %v", err)
	}
}

//electionHandler returns the status whether the cloudcore is ready
func electionHandler(w http.ResponseWriter, r *http.Request) {
	checker := hubconfig.Config.Checker
	if checker == nil {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Cloudcore is ready with no leaderelection")); err != nil {
			klog.Errorf("failed to write http response, err: %v", err)
		}
		return
	}
	if checker.Check(r) != nil {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("Cloudcore is not ready")); err != nil {
			klog.Errorf("failed to write http response, err: %v", err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Cloudcore is ready")); err != nil {
			klog.Errorf("failed to write http response, err: %v", err)
		}
	}
}

// EncodeCertPEM returns PEM-endcoded certificate data
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  certificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// edgeCoreClientCert will verify the certificate of EdgeCore or token then create EdgeCoreCert and return it
func edgeCoreClientCert(w http.ResponseWriter, r *http.Request) {
	if cert := r.TLS.PeerCertificates; len(cert) > 0 {
		if err := verifyCert(cert[0]); err != nil {
			klog.Errorf("failed to sign the certificate for edgenode: %s, failed to verify the certificate", r.Header.Get(constants.NodeName))
			w.WriteHeader(http.StatusUnauthorized)
			if _, err := w.Write([]byte(err.Error())); err != nil {
				klog.Errorf("failed to write response, err: %v", err)
			}
		} else {
			signEdgeCert(w, r)
		}
		return
	}
	if verifyAuthorization(w, r) {
		signEdgeCert(w, r)
	} else {
		klog.Errorf("failed to sign the certificate for edgenode: %s, invalid token", r.Header.Get(constants.NodeName))
	}
}

// verifyCert verifies the edge certificate by CA certificate when edge certificates rotate.
func verifyCert(cert *x509.Certificate) error {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: certificateBlockType, Bytes: hubconfig.Config.Ca}))
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
				klog.Errorf("Wrire body error %v", err)
			}
		}
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("Invalid authorization token")); err != nil {
			klog.Errorf("Wrire body error %v", err)
		}

		return false
	}
	if !token.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte("Invalid authorization token")); err != nil {
			klog.Errorf("Wrire body error %v", err)
		}
		return false
	}
	return true
}

// signEdgeCert signs the CSR from EdgeCore
func signEdgeCert(w http.ResponseWriter, r *http.Request) {
	csrContent, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("fail to read file when signing the cert for edgenode:%s! error:%v", r.Header.Get(constants.NodeName), err)
	}
	csr, err := x509.ParseCertificateRequest(csrContent)
	if err != nil {
		klog.Errorf("fail to ParseCertificateRequest of edgenode: %s! error:%v", r.Header.Get(constants.NodeName), err)
	}
	subject := csr.Subject
	clientCertDER, err := signCerts(subject, csr.PublicKey)
	if err != nil {
		klog.Errorf("fail to signCerts for edgenode:%s! error:%v", r.Header.Get(constants.NodeName), err)
	}

	if _, err := w.Write(clientCertDER); err != nil {
		klog.Errorf("wrire error %v", err)
	}
}

// signCerts will create a certificate for EdgeCore
func signCerts(subInfo pkix.Name, pbKey crypto.PublicKey) ([]byte, error) {
	cfgs := &certutil.Config{
		CommonName:   subInfo.CommonName,
		Organization: subInfo.Organization,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
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

// CheckCaExistsFromSecret checks ca from secret
func CheckCaExistsFromSecret() bool {
	if _, err := GetSecret(CaSecretName, NamespaceSystem); err != nil {
		return false
	}
	return true
}

// CheckCertExistsFromSecret checks CloudCore certificate from secret
func CheckCertExistsFromSecret() bool {
	if _, err := GetSecret(CloudCoreSecretName, NamespaceSystem); err != nil {
		return false
	}
	return true
}

// PrepareAllCerts check whether the certificates exist in the local directory,
// and then check whether certificates exist in the secret, generate if they don't exist
func PrepareAllCerts() error {
	// Check whether the ca exists in the local directory
	if hubconfig.Config.Ca == nil && hubconfig.Config.CaKey == nil {
		klog.Info("Ca and CaKey don't exist in local directory, and will read from the secret")
		// Check whether the ca exists in the secret
		secretHasCA := CheckCaExistsFromSecret()
		if !secretHasCA {
			klog.Info("Ca and CaKey don't exist in the secret, and will be created by CloudCore")
			caDER, caKey, err := NewCertificateAuthorityDer()
			if err != nil {
				klog.Errorf("failed to create Certificate Authority, error: %v", err)
				return err
			}

			caKeyDER, err := x509.MarshalECPrivateKey(caKey.(*ecdsa.PrivateKey))
			if err != nil {
				klog.Errorf("failed to convert an EC private key to SEC 1, ASN.1 DER form, error: %v", err)
				return err
			}

			err = CreateCaSecret(caDER, caKeyDER)
			if err != nil {
				klog.Errorf("failed to create ca to secrets, error: %v", err)
				return err
			}

			UpdateConfig(caDER, caKeyDER, nil, nil)
		} else {
			s, err := GetSecret(CaSecretName, NamespaceSystem)
			if err != nil {
				klog.Errorf("failed to get CaSecret, error: %v", err)
				return err
			}
			caDER := s.Data[CaDataName]
			caKeyDER := s.Data[CaKeyDataName]

			UpdateConfig(caDER, caKeyDER, nil, nil)
		}
	} else {
		// HubConfig has been initialized
		ca := hubconfig.Config.Ca
		caKey := hubconfig.Config.CaKey
		err := CreateCaSecret(ca, caKey)
		if err != nil {
			klog.Errorf("failed to save ca and key to the secret, error: %v", err)
			return err
		}
	}

	// Check whether the CloudCore certificates exist in the local directory
	if hubconfig.Config.Key == nil && hubconfig.Config.Cert == nil {
		klog.Infof("CloudCoreCert and key don't exist in local directory, and will read from the secret")
		// Check whether the CloudCore certificates exist in the secret
		secretHasCert := CheckCertExistsFromSecret()
		if !secretHasCert {
			klog.Info("CloudCoreCert and key don't exist in the secret, and will be signed by CA")
			certDER, keyDER, err := SignCerts()
			if err != nil {
				klog.Errorf("failed to sign a certificate, error: %v", err)
				return err
			}

			err = CreateCloudCoreSecret(certDER, keyDER)
			if err != nil {
				klog.Errorf("failed to save CloudCore cert and key to secret, error: %v", err)
				return err
			}

			UpdateConfig(nil, nil, certDER, keyDER)
		} else {
			s, err := GetSecret(CloudCoreSecretName, NamespaceSystem)
			if err != nil {
				klog.Errorf("failed to get CloudCore secret, error: %v", err)
				return err
			}
			certDER := s.Data[CloudCoreCertName]
			keyDER := s.Data[CloudCoreKeyDataName]

			UpdateConfig(nil, nil, certDER, keyDER)
		}
	} else {
		// HubConfig has been initialized
		cert := hubconfig.Config.Cert
		key := hubconfig.Config.Key
		err := CreateCloudCoreSecret(cert, key)
		if err != nil {
			klog.Errorf("failed to save CloudCore cert to secret, error: %v", err)
			return err
		}
	}
	return nil
}
