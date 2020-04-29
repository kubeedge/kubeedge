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
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"strings"

	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/common/constants"
	utilvalidation "github.com/kubeedge/kubeedge/pkg/util/validation"
)

// StartHttpServer starts the http service
func StartHttpServer() {
	router := mux.NewRouter()
	router.HandleFunc("/edge.crt", edgeCoreClientCert).Methods("GET")
	router.HandleFunc("/ca.crt", getCA).Methods("GET")

	addr := fmt.Sprintf("%s:%d", hubconfig.Config.Https.Address, hubconfig.Config.Https.Port)
	klog.Fatal(http.ListenAndServeTLS(addr, "", "", router))
}

// getCA returns the caCertDER
func getCA(w http.ResponseWriter, r *http.Request) {
	caCertDER := hubconfig.Config.Ca
	w.Write(caCertDER)
}

// edgeCoreClientCert will verify the token then create EdgeCoreCert and return it
func edgeCoreClientCert(w http.ResponseWriter, r *http.Request) {
	authorizationHeader := r.Header.Get("authorization")
	if authorizationHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("Invalid authorization token")))
		return
	}
	bearerToken := strings.Split(authorizationHeader, " ")
	if len(bearerToken) != 2 {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("Invalid authorization token")))
		return
	}
	token, err := jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("There was an error")
		}
		caKey := hubconfig.Config.CaKey
		return caKey, nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(fmt.Sprintf("Invalid authorization token")))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Invalid authorization token")))
		return
	}
	if !token.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("Invalid authorization token")))
		return
	}

	csrContent, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Errorf("fail to read file! error:%v", err)
	}
	csr, err := x509.ParseCertificateRequest(csrContent)
	if err != nil {
		fmt.Errorf("fail to ParseCertificateRequest! error:%v", err)
	}
	subject := csr.Subject
	clientCertDER, err := signCerts(subject, csr.PublicKey)
	if err != nil {
		fmt.Errorf("fail to signCerts! error:%v", err)
	}

	w.Write(clientCertDER)
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
	caKey, err := x509.ParsePKCS1PrivateKey(caKeyDER)
	if err != nil {
		return nil, fmt.Errorf("unable to ParsePKCS1PrivateKey: %v", err)
	}

	certDER, err := NewCertFromCa(cfgs, caCert, clientKey, crypto.Signer(caKey))
	if err != nil {
		return nil, fmt.Errorf("unable to NewCertFromCa: %v", err)
	}

	return certDER, err
}

func CheckCaExistsFromSecret() bool {
	if _, err := GetSecret(CaSecretName, NamespaceSystem); err != nil {
		return false
	}
	return true

}

func CheckCertExistsFromSecret() bool {
	if _, err := GetSecret(CloudCoreSecretName, NamespaceSystem); err != nil {
		return false
	}
	return true
}

// PrepareAllCerts check whether the certificates exist in the local directory,
// and then check whether certificates exist in the secret, generate if they don't exist
func PrepareAllCerts() {
	// Check whether the ca exists in the local directory
	if !(utilvalidation.FileIsExist(hubconfig.Config.CloudHub.TLSCAFile) && utilvalidation.FileIsExist(hubconfig.Config.CloudHub.TLSCAKeyFile)) {
		// Check whether the ca exists in the secret
		secretHasCA := CheckCaExistsFromSecret()
		if !secretHasCA {
			caDER, caKey, err := NewCertificateAuthorityDer()
			if err != nil {
				klog.Errorf("failed to create Certificate Authority, error: %v", err)
			}

			caKeyDER := x509.MarshalPKCS1PrivateKey(caKey.(*rsa.PrivateKey))

			CreateCaSecret(caDER, caKeyDER)

			UpdateConfig(caDER, caKeyDER, []byte(""), []byte(""))
		} else {
			s, err := GetSecret(CaSecretName, NamespaceSystem)
			if err != nil {
				klog.Errorf("failed to get CaSecret, error: %v", err)
				fmt.Errorf("failed to get CaSecret, error: %v", err)
			}
			caDER := s.Data[CaDataName]
			caKeyDER := s.Data[CaKeyDataName]

			UpdateConfig(caDER, caKeyDER, []byte(""), []byte(""))
		}
	} else {
		// HubConfig has been initialized
		ca := hubconfig.Config.Ca
		caKey := hubconfig.Config.CaKey
		CreateCaSecret(ca, caKey)
	}

	// Check whether the CloudCore certificates exist in the local directory
	if !(utilvalidation.FileIsExist(hubconfig.Config.CloudHub.TLSCertFile) && utilvalidation.FileIsExist(hubconfig.Config.CloudHub.TLSPrivateKeyFile)) {
		klog.Errorf("TLSCertFile and TLSPrivateKeyFile don't exist")
		fmt.Println("TLSCertFile and TLSPrivateKeyFile don't git reset --soft HEAD^exist")
		// Check whether the CloudCore certificates exist in the secret
		secretHasCert := CheckCertExistsFromSecret()
		if !secretHasCert {
			certDER, keyDER := SignCerts()

			CreateCloudCoreSecret(certDER, keyDER)

			UpdateConfig([]byte(""), []byte(""), certDER, keyDER)
		} else {
			s, err := GetSecret(CloudCoreSecretName, NamespaceSystem)
			if err != nil {
				klog.Errorf("failed to get cloudcore secret, error: %v", err)
				fmt.Errorf("failed to get cloudcore secret error: %v", err)
			}
			certDER := s.Data[CloudCoreDataName]
			keyDER := s.Data[CloudCoreKeyDataName]

			UpdateConfig([]byte(""), []byte(""), certDER, keyDER)
		}
	} else {
		// HubConfig has been initialized
		cert := hubconfig.Config.Cert
		key := hubconfig.Config.Key
		CreateCaSecret(cert, key)
	}
}
