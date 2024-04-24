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

package restful

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	util2 "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/util"
	keadutil "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	CoreAPIPrefix       = "api"
	CoreAPIGroupVersion = schema.GroupVersion{Group: "", Version: "v1"}
	Prefix              = "apis"
)

type Request struct {
	Method string
	Path   string
	Body   io.Reader
}

func (req *Request) RestfulRequest() (*http.Response, error) {
	var client http.Client
	config, err := keadutil.ParseEdgecoreConfig(common.EdgecoreConfigPath)
	if err != nil {
		return nil, fmt.Errorf("get edge config failed with err:%v", err)
	}
	if config.Modules.MetaManager.MetaServer.Enable {
		url := config.Modules.MetaManager.MetaServer.Server
		ok, requireAuthorization := config.FeatureGates["requireAuthorization"]
		if ok && requireAuthorization {
			serverCrt := config.Modules.MetaManager.MetaServer.TLSCertFile
			serverKey := config.Modules.MetaManager.MetaServer.TLSPrivateKeyFile
			cert, err := tls.LoadX509KeyPair(serverCrt, serverKey)
			if err != nil {
				return nil, fmt.Errorf("failed to load server certificate and private key with err:%v", err)
			}

			tlsCaFile := config.Modules.MetaManager.MetaServer.TLSCaFile
			caCert, err := os.ReadFile(tlsCaFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load tlsCaFile with err:%v", err)
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{cert},
				//ClientAuth:   tls.RequireAndVerifyClientCert,
				RootCAs: caCertPool,
			}

			url = "https://" + url
			client = http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsConfig,
				},
			}
		} else {
			url = "http://" + url
			client = http.Client{}
		}

		request, err := http.NewRequest(req.Method, url+req.Path, req.Body)
		if err != nil {
			return nil, fmt.Errorf("restful format failed with err:%v", err)
		}
		response, err := client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("restful failed with err:%v", err)
		}

		return response, nil
	}
	return nil, fmt.Errorf("metaserver don't open")
}

func (req *Request) ResponseToPodList() (*corev1.PodList, error) {
	response, err := req.RestfulRequest()
	if err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read response's body failed with err:%v", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, util2.GetErrMessage(bodyBytes)
	}

	var podList *corev1.PodList
	err = json.Unmarshal(bodyBytes, &podList)
	if err != nil {
		return nil, fmt.Errorf("parsing response's body failed with err:%v", err)
	}

	return podList, err
}

func (req *Request) ResponseToPod() (*corev1.Pod, error) {
	response, err := req.RestfulRequest()
	if err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read response's body failed with err:%v", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, util2.GetErrMessage(bodyBytes)
	}

	var pod *corev1.Pod
	err = json.Unmarshal(bodyBytes, &pod)
	if err != nil {
		return nil, fmt.Errorf("parsing response's body failed with err:%v", err)
	}

	return pod, err
}
