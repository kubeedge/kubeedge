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

package v1alpha1

import (
	cloudcoreconfig "github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/apis/edgecore/v1alpha1"
	metaconfig "github.com/kubeedge/kubeedge/pkg/apis/meta/v1alpha1"
)

type EdgeSideConfig struct {
	metaconfig.TypeMeta
	// Mqtt set mqtt config for edgesite, shared with edgecore mqttconfig
	// +Required
	Mqtt edgecoreconfig.MqttConfig `json:"mqtt,omitempty"`
	// Kube set kubernetes cluster info which will be connect, shared with cloudcore kubeconfig
	// +Required
	Kube cloudcoreconfig.KubeConfig `json:"kube,omitempty"`
	// ControllerContext set controller context ,shared with cloudcore controller context
	// +Required
	ControllerContext cloudcoreconfig.ControllerContext `json:"controllerContext"`
	// Edged set edged module config,shared with edgecore edged config
	// +Required
	Edged edgecoreconfig.EdgedConfig `json:"edged,omitempty"`
	// Modules set which modules are enabled
	// +Required
	Modules metaconfig.Modules `json:"modules,omitempty"`
	// +Required
	// set meta module config ,shared with edgecore Metamanager config
	MetaManager edgecoreconfig.MetaManager `json:"metaManager,omitempty"`
	// DataBase set DataBase config
	// +Required
	DataBase edgecoreconfig.DataBase `json:"database,omitempty"`
}
