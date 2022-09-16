/*
Copyright 2021 The KubeEdge Authors.

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
	"os"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	cloudcoreConfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
)

var (
	initOnce      sync.Once
	kubeClient    kubernetes.Interface
	crdClient     crdClientset.Interface
	dynamicClient dynamic.Interface
	// authKubeConfig only contains master address and CA cert when init, it is used for
	// generating a temporary kubeclient and validating user token once receive an application message.
	authKubeConfig *rest.Config
)

func InitKubeEdgeClient(config *cloudcoreConfig.KubeAPIConfig) {
	initOnce.Do(func() {
		kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Master, config.KubeConfig)
		if err != nil {
			klog.Errorf("Failed to build config, err: %v", err)
			os.Exit(1)
		}
		kubeConfig.QPS = float32(config.QPS)
		kubeConfig.Burst = int(config.Burst)

		dynamicClient = dynamic.NewForConfigOrDie(kubeConfig)

		kubeConfig.ContentType = runtime.ContentTypeProtobuf
		kubeClient = kubernetes.NewForConfigOrDie(kubeConfig)

		crdKubeConfig := rest.CopyConfig(kubeConfig)
		crdKubeConfig.ContentType = runtime.ContentTypeJSON
		crdClient = crdClientset.NewForConfigOrDie(crdKubeConfig)

		authKubeConfig, err = clientcmd.BuildConfigFromFlags(kubeConfig.Host, "")
		if err != nil {
			klog.Errorf("Failed to build config, err: %v", err)
			os.Exit(1)
		}
		authKubeConfig.CAData = kubeConfig.CAData
		authKubeConfig.CAFile = kubeConfig.CAFile
		authKubeConfig.ContentType = runtime.ContentTypeJSON
	})
}

func GetKubeClient() kubernetes.Interface {
	return kubeClient
}

func GetCRDClient() crdClientset.Interface {
	return crdClient
}

func GetDynamicClient() dynamic.Interface {
	return dynamicClient
}

func GetAuthConfig() *rest.Config {
	return authKubeConfig
}
