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
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	certutil "k8s.io/client-go/util/cert"
)


//create server's certificate and key
func SignCerts()([]byte,[]byte) {
	cfgs := &certutil.Config{
		CommonName:   "kubeedge",
		Organization: []string{"HuaWei"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		AltNames:     nil,
	}
	//new server's certificate from ca
	certDER,keyDER,err:=NewCloudCoreCertDERandKey(cfgs)
	if err!=nil{
		fmt.Printf("%v",err)
	}
	return certDER,keyDER
}


func GenerateToken() {
	expiresAt := time.Now().Add(time.Hour * 24).Unix()

	token := jwt.New(jwt.SigningMethodHS256)

	token.Claims = jwt.StandardClaims{
		ExpiresAt: expiresAt,
	}

	keyPEM := getCaKey()
	tokenString, err := token.SignedString(keyPEM)

	if err!=nil{
		fmt.Printf("%v",err)
	}
	caHash :=getCahash()
	//combine caHash and tokenString into caHashAndToken
	caHashAndToken:=strings.Join([]string{caHash, tokenString}, " ")
	//save caHashAndToken to secret
	CreateTokenSecret([]byte(caHashAndToken))

	fmt.Println(caHashAndToken)

	t := time.NewTicker(time.Hour * 12)
	go func() {
		for {
			select {
			case <-t.C:
				refreshToken()
			}
		}
	}()
}



func refreshToken() string {
	claims := &jwt.StandardClaims{}
	expirationTime := time.Now().Add(5 * time.Minute)
	claims.ExpiresAt = expirationTime.Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	keyPEM := getCaKey()
	tokenString, _ := token.SignedString(keyPEM)
	caHash :=getCahash()
	//put caHash in token
	caHashAndToken:=strings.Join([]string{caHash, tokenString}, " ")
	return caHashAndToken
}

// getCahash get ca-hash
func getCahash() string {
	caDER:= hubconfig.Config.Ca
	digest:=sha256.Sum256(caDER)
	return hex.EncodeToString(digest[:])
}

//getCaKey get caKey to encrypt token
func getCaKey() []byte {
	caKey:=hubconfig.Config.CaKey
	return caKey
}
