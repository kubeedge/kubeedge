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
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	keadutil "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func KubeClient() (*kubernetes.Clientset, error) {
	kubeConfig, err := GetKubeConfig()
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

func GetKubeConfig() (*restclient.Config, error) {
	config, err := keadutil.ParseEdgecoreConfig(common.EdgecoreConfigPath)
	if err != nil {
		return nil, fmt.Errorf("get edge config failed with err:%v", err)
	}

	if !config.Modules.MetaManager.MetaServer.Enable {
		return nil, fmt.Errorf("metaserver don't open")
	}

	url := config.Modules.MetaManager.MetaServer.Server
	ok, requireAuthorization := config.FeatureGates["requireAuthorization"]
	if ok && requireAuthorization {
		url = "https://" + url
	} else {
		url = "http://" + url
	}
	kubeConfig, err := clientcmd.BuildConfigFromFlags(url, "")
	if err != nil {
		return nil, err
	}

	if ok && requireAuthorization {
		serverCrt := config.Modules.MetaManager.MetaServer.TLSCertFile
		serverKey := config.Modules.MetaManager.MetaServer.TLSPrivateKeyFile
		tlsCaFile := config.Modules.MetaManager.MetaServer.TLSCaFile

		kubeConfig.TLSClientConfig.CAFile = tlsCaFile
		kubeConfig.TLSClientConfig.CertFile = serverCrt
		kubeConfig.TLSClientConfig.KeyFile = serverKey
	}
	kubeConfig.Timeout = 1 * time.Minute
	return kubeConfig, nil
}

func VersionedKubeClient() (*versioned.Clientset, error) {
	kubeConfig, err := GetKubeConfig()
	if err != nil {
		return nil, err
	}
	versionedClient, err := versioned.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	return versionedClient, nil
}
