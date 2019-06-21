/*
Copyright 2018 The KubeEdge Authors.

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

package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/kubeedge/beehive/pkg/common/config"
	"gopkg.in/yaml.v2"
)

// LoadConfig is function to Load Configurations from a specified location. If no location is specified it loads the config from the default location
func LoadConfig(confLocation ...string) error {
	err := config.CONFIG.DeInit()
	if err != nil {
		return err
	}
	// TODO(fisherxu): change to use modules in go-chassis
	fSource := NewYamlConfigurationSource()
	if len(confLocation) == 0 {
		confLocation = []string{os.Getenv("GOPATH") + "/src/github.com/kubeedge/kubeedge/edge/conf"}
	}
	err = filepath.Walk(confLocation[0], func(location string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		ext := strings.ToLower(path.Ext(location))
		if ext == ".yml" || ext == ".yaml" {
			fSource.AddFileSource(location, 0)
		}
		return nil
	})
	if err != nil {
		return err
	}
	config.CONFIG.AddSource(fSource)
	return nil
}

//GenerateTestYaml is a function is used to create a temporary file to be used for testing
//It accepts 3 arguments:"test" is the interface used to generate the YAML,
// "path" is the directory path at which the directory is to be created,
// "filename" is the name of the file to be created without the ".yaml" extension
func GenerateTestYaml(test interface{}, path, filename string) error {
	data, err := yaml.Marshal(test)
	if err != nil {
		return err
	}
	err = os.MkdirAll(path, 0777)
	if err != nil {
		return err
	}
	fp, err := os.Create(path + "/" + filename + ".yaml")
	if err != nil {
		return err
	}
	_, err = fp.Write(data)
	if err != nil {
		return err
	}
	if err = fp.Sync(); err != nil {
		return err
	}
	if err = fp.Close(); err != nil {
		return err
	}
	return nil
}

//GenerateTestCertificate generates fake certificates and stores them at the path specified.
//It accepts 3 arguments path, certFileName and keyFileName
// "path" is the directory path at which the directory is to be created,
// "certFileName" & "keyFileName" refers to the name of the file to be created without the extension
func GenerateTestCertificate(path string, certFileName string, keyFileName string) error {
	template := &x509.Certificate{
		IsCA:                  true,
		BasicConstraintsValid: true,
		SubjectKeyId:          []byte{1, 2, 3},
		SerialNumber:          big.NewInt(1234),
		Subject: pkix.Name{
			Country:      []string{"test"},
			Organization: []string{"testor"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(5, 5, 5),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
	// generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	publicKey := &privateKey.PublicKey
	// create a self-signed certificate. template = parent
	var parent = template
	cert, err := x509.CreateCertificate(rand.Reader, template, parent, publicKey, privateKey)
	if err != nil {
		return err
	}
	err = os.MkdirAll(path, 0777)
	if err != nil {
		return err
	}
	pKey := x509.MarshalPKCS1PrivateKey(privateKey)
	certFilePEM := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert}
	err = createPEMfile(path+certFileName+".crt", certFilePEM)
	if err != nil {
		return err
	}
	keyFilePEM := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pKey}
	err = createPEMfile(path+keyFileName+".key", keyFilePEM)
	if err != nil {
		return err
	}
	return nil
}

//createPEMfile() creates an encoded file at the path given, with PEM Block specified
func createPEMfile(path string, pemBlock pem.Block) error {
	// this will create plain text PEM file.
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer file.Close()
	err = pem.Encode(file, &pemBlock)
	return err
}
