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

package config

import (
	"encoding/pem"
	"io/ioutil"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.CloudStream
	Ca   []byte
	Cert []byte
	Key  []byte
}

func InitConfigure(stream *v1alpha1.CloudStream) {
	once.Do(func() {
		Config = Configure{
			CloudStream: *stream,
		}

		ca, err := ioutil.ReadFile(stream.TLSTunnelCAFile)
		if err == nil {
			block, _ := pem.Decode(ca)
			ca = block.Bytes
		}
		if ca != nil {
			Config.Ca = ca
		}

		cert, err := ioutil.ReadFile(stream.TLSTunnelCertFile)
		if err == nil {
			block, _ := pem.Decode(cert)
			cert = block.Bytes
		}

		key, err := ioutil.ReadFile(stream.TLSTunnelPrivateKeyFile)
		if err == nil {
			block, _ := pem.Decode(key)
			key = block.Bytes
		}

		if cert != nil && key != nil {
			Config.Cert = cert
			Config.Key = key
		} else if !(cert == nil && key == nil) {
			klog.Fatal("Both of tunnelCert and key should be specified!")
		}
	})
}
