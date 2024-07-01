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

package client

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
	kefeatures "github.com/kubeedge/kubeedge/pkg/features"
)

var kubeClient *kubernetes.Clientset
var once sync.Once

func InitKubeClient() {
	once.Do(func() {
		kubeConfig, err := getKubeConfig()
		if err != nil {
			klog.Errorf("get kube config err: %v", err)
			return
		}
		kubeClient, err = kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			klog.Errorf("init kubeClient err: %v", err)
		}
	})
}

func getKubeConfig() (*restclient.Config, error) {
	if !metaserverconfig.Config.MetaServer.Enable {
		return nil, errors.New("metaserver don't open")
	}
	url := metaserverconfig.Config.Server
	if kefeatures.DefaultFeatureGate.Enabled(kefeatures.RequireAuthorization) {
		if !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}
	} else {
		if !strings.HasPrefix(url, "http://") {
			url = "http://" + url
		}
	}
	kubeConfig, err := clientcmd.BuildConfigFromFlags(url, "")
	if err != nil {
		return nil, err
	}

	if kefeatures.DefaultFeatureGate.Enabled(kefeatures.RequireAuthorization) {
		serverCrt := metaserverconfig.Config.TLSCertFile
		serverKey := metaserverconfig.Config.TLSPrivateKeyFile
		tlsCaFile := metaserverconfig.Config.TLSCaFile
		kubeConfig.TLSClientConfig.CAFile = tlsCaFile
		kubeConfig.TLSClientConfig.CertFile = serverCrt
		kubeConfig.TLSClientConfig.KeyFile = serverKey
	}
	kubeConfig.Timeout = 10 * time.Second
	return kubeConfig, nil
}

func GetKubeClient() (*kubernetes.Clientset, error) {
	if kubeClient != nil {
		return kubeClient, nil
	}
	return nil, fmt.Errorf("please check if metaserver is opened")
}
