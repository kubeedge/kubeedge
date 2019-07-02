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

// Package utils
package utils

import (
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/config"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeConfig from flags
func KubeConfig() (conf *rest.Config, err error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Kube.KubeMaster, config.Kube.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = config.Kube.KubeQPS
	kubeConfig.Burst = config.Kube.KubeBurst
	kubeConfig.ContentType = config.Kube.KubeContentType

	return kubeConfig, nil
}
