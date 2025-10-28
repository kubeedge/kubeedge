/*
Copyright 2025 The KubeEdge Authors.

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

package config

import (
	"encoding/pem"
	"os"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

var (
	Config Configure
	once   sync.Once
)

type Configure struct {
	v1alpha1.CloudDataStream
	Ca   []byte
	Cert []byte
	Key  []byte
}

func InitConfigure(stream *v1alpha1.CloudDataStream) {
	once.Do(func() {
		Config = Configure{
			CloudDataStream: *stream,
		}

		ca, err := os.ReadFile(stream.TLSTunnelCAFile)
		if err == nil {
			block, _ := pem.Decode(ca)
			if block == nil {
				klog.Warningf("Failed to decode PEM block from ca file: %s", stream.TLSTunnelCAFile)
				ca = nil
			} else {
				ca = block.Bytes
			}
		}
		if ca != nil {
			Config.Ca = ca
		}

		cert, err := os.ReadFile(stream.TLSTunnelCertFile)
		if err == nil {
			block, _ := pem.Decode(cert)
			if block == nil {
				klog.Warningf("Failed to decode PEM block from cert file: %s", stream.TLSTunnelCertFile)
				cert = nil
			} else {
				cert = block.Bytes
			}
		}

		key, err := os.ReadFile(stream.TLSTunnelPrivateKeyFile)
		if err == nil {
			block, _ := pem.Decode(key)
			if block == nil {
				klog.Warningf("Failed to decode PEM block from key file: %s", stream.TLSTunnelPrivateKeyFile)
				key = nil
			} else {
				key = block.Bytes
			}
		}

		if cert != nil && key != nil {
			Config.Cert = cert
			Config.Key = key
		} else if !(cert == nil && key == nil) {
			klog.Exit("Both of tunnelCert and key should be specified!")
		}
	})
}
