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
	"fmt"
	"os"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	cloudcoreConfig "github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	crdClientset "github.com/kubeedge/api/client/clientset/versioned"
)

var (
	initOnce      sync.Once
	kubeClient    kubernetes.Interface
	crdClient     crdClientset.Interface
	dynamicClient dynamic.Interface

	KubeConfig *rest.Config
	CrdConfig  *rest.Config
)

// InitKubeEdgeClient initializes the shared Kubernetes clients (kube, CRD, and
// dynamic) used throughout the cloud components. It is safe to call multiple
// times; subsequent calls are no-ops. The function panics if the kube config
// cannot be built from the provided KubeAPIConfig.
func InitKubeEdgeClient(config *cloudcoreConfig.KubeAPIConfig, enableImpersonation bool) {
	initOnce.Do(func() {
		kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Master, config.KubeConfig)
		if err != nil {
			panic(fmt.Errorf("failed to build kube config, err: %v", err))
		}
		kubeConfig.QPS = float32(config.QPS)
		kubeConfig.Burst = int(config.Burst)

		KubeConfig = kubeConfig

		dynamicClient = newForDynamicConfigOrDie(kubeConfig, enableImpersonation)

		kubeConfig.ContentType = runtime.ContentTypeProtobuf
		kubeClient = newForK8sConfigOrDie(kubeConfig, enableImpersonation)

		crdKubeConfig := rest.CopyConfig(kubeConfig)
		crdKubeConfig.ContentType = runtime.ContentTypeJSON
		CrdConfig = crdKubeConfig
		crdClient = newForCrdConfigOrDie(crdKubeConfig, enableImpersonation)
	})
}

// GetKubeClient returns the shared Kubernetes client for built-in API resources.
func GetKubeClient() kubernetes.Interface {
	return kubeClient
}

// GetCRDClient returns the shared client for KubeEdge custom resource definitions.
func GetCRDClient() crdClientset.Interface {
	return crdClient
}

// GetDynamicClient returns the shared dynamic client used for third-party CRD resources.
func GetDynamicClient() dynamic.Interface {
	return dynamicClient
}

// GetK8sCA reads and returns the Kubernetes CA certificate bytes from the
// path specified in KubeConfig. Returns nil and logs an error if the file
// cannot be read.
func GetK8sCA() []byte {
	ca, err := os.ReadFile(KubeConfig.CAFile)
	if err != nil {
		klog.Errorf("read k8s CA failed, %v", err)
		return nil
	}
	return ca
}

// RestMapperFunc is a function type that returns a REST mapper for mapping
// GroupVersionKinds to API resources.
type RestMapperFunc func() (meta.RESTMapper, error)

var DefaultGetRestMapper RestMapperFunc = GetRestMapper

// GetRestMapper returns a dynamic REST mapper built from the current
// KubeConfig. The mapper is used to resolve GroupVersionResource mappings
// at runtime without requiring the API groups to be known at compile time.
func GetRestMapper() (meta.RESTMapper, error) {
	client, err := rest.HTTPClientFor(KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("new http client for kubeConfig failed, err: %v", err)
	}
	return apiutil.NewDynamicRESTMapper(KubeConfig, client)
}
