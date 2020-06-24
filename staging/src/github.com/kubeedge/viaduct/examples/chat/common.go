/*
Copyright 2019 The KubeEdge Authors.

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

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"k8s.io/klog"

	"github.com/kubeedge/viaduct/examples/chat/config"
)

func GetTlsConfig(cfg *config.Config) (*tls.Config, error) {
	if cfg.CaFile == "" ||
		cfg.CertFile == "" ||
		cfg.KeyFile == "" {
		return nil, fmt.Errorf("bad cert certification files")
	}

	caBytes, err := ioutil.ReadFile(cfg.CaFile)
	if err != nil {
		klog.Errorf("failed to read ca file(%s), error: %+v", cfg.CaFile, err)
		return nil, err
	}
	cerBytes, err := ioutil.ReadFile(cfg.CertFile)
	if err != nil {
		klog.Errorf("failed to read cert file(%s), error: %+v", cfg.CertFile, err)
		return nil, err
	}
	keyBytes, err := ioutil.ReadFile(cfg.KeyFile)
	if err != nil {
		klog.Errorf("failed to read key file(%s), error: %+v", cfg.KeyFile, err)
		return nil, err
	}

	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(caBytes)
	if !ok {
		klog.Error("failed to append certs")
		return nil, fmt.Errorf("failed to append certs")
	}
	cert, err := tls.X509KeyPair(cerBytes, keyBytes)
	if err != nil {
		klog.Errorf("failed to get key pair, error: %+v", err)
		return nil, err
	}

	return &tls.Config{
		ClientCAs:    pool,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}
