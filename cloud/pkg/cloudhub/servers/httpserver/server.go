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
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog"
)

// StartHttpServer starts the http service
func StartHttpServer() {
	router := mux.NewRouter()
	router.HandleFunc("/edge.crt", edgeCoreClientCert).Methods("GET")
	//router.HandleFunc("/client.crt", edgeCoreClientCert).Methods("GET")
	router.HandleFunc("/ca.crt", getCA).Methods("GET")

	klog.Fatal(http.ListenAndServeTLS(":3000", "", "", router))
}

//done
func getCA(w http.ResponseWriter, r *http.Request) {
	caCertDER := hubconfig.Config.Ca
	w.Write(caCertDER) //w.Write([]byte(fmt.Sprintf("CA will be returned")))

}

//edgeCoreClientCert will verify the token then create edgeCoreCert and return it
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
		//return []byte("secret"), nil
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

	csrContent, _ := ioutil.ReadAll(r.Body)
	csr, _ := x509.ParseCertificateRequest(csrContent)
	subject := csr.Subject
	// sign the certs using CA and return to edge
	clientCertDER, err := signCerts(subject, csr.PublicKey)
	w.Write(clientCertDER)
	//w.Write([]byte(fmt.Sprintf("Will return the certs for edgecore")))
}

//signCerts will create a certificate for EdgeCore
func signCerts(subInfo pkix.Name, pbKey crypto.PublicKey) ([]byte, error) {
	cfgs := &certutil.Config{
		CommonName:   subInfo.CommonName,
		Organization: subInfo.Organization,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientKey := pbKey

	//get ca from config
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

func CheckCAExists() bool {
	if _, err := GetSecret(CaSecretName, NamespaceSystem); err == nil {
		return true
	} else {
		return false
	}
}
func CheckCloudCoreCertExists() bool {
	if _, err := GetSecret(CaSecretName, NamespaceSystem); err == nil {
		return true
	} else {
		return false
	}
}

func PrepareForAllCert() {
	//check if there exists ca in secret
	hasCA := CheckCAExists()
	//generate ca if not find caCert
	if !hasCA {
		caDER, caKey, _ := NewCertificateAuthorityDer()
		caCert, err := x509.ParseCertificate(caDER)
		if err != nil {
			fmt.Printf("%v", err)
		}

		WriteCertAndKey("/etc/kubeedge/ca/", "rootCA", caCert, caKey)

		caKeyDER := x509.MarshalPKCS1PrivateKey(caKey.(*rsa.PrivateKey))

		CreateCaSecret(caDER, caKeyDER)

		UpdateConfig(caDER, caKeyDER, []byte(""), []byte(""))

	} else {
		s, err := GetSecret(CaSecretName, NamespaceSystem)
		if err != nil {
			fmt.Printf("%v", err)
		}
		caDER := s.Data[CaDataName]
		caKeyDER := s.Data[CaKeyDataName]

		UpdateConfig(caDER, caKeyDER, []byte(""), []byte(""))
	}

	hasCloudCoreCert := CheckCloudCoreCertExists()
	//generate ca if not find caCert
	if !hasCloudCoreCert {
		//The previous step ensures that this SignCerts can be performed because there exists ca
		certDER, keyDER := SignCerts()

		cert, key, err := ParseCertDerToCertificate(certDER, keyDER)
		if err != nil {
			fmt.Printf("%v", err)
		}

		//save it to secret(Etcd)
		CreateCloudCoreSecret(certDER, keyDER)

		//Save cloudCoreCert file to filePath, Note:although this is cloudCoreCert,but it's called edge.crt/edge.key
		WriteCertAndKey("/etc/kubeedge/certs/", "edge", cert, key)

		UpdateConfig([]byte(""), []byte(""), certDER, keyDER)
	} else {
		s, err := GetSecret(CloudCoreSecretName, NamespaceSystem)
		if err != nil {
			fmt.Printf("%v", err)
		}
		certDER := s.Data[CloudCoreDataName]
		keyDER := s.Data[CloudCoreKeyDataName]
		UpdateConfig([]byte(""), []byte(""), certDER, keyDER)
	}
}
