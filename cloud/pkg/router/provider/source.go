/*
Copyright 2022 The KubeEdge Authors.

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

package provider

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	v1 "github.com/kubeedge/kubeedge/pkg/apis/rules/v1"
)

type SourceFactory interface {
	Type() v1.RuleEndpointTypeDef
	GetSource(ep *v1.RuleEndpoint, sourceResource map[string]string) Source
}

type Source interface {
	Name() string
	RegisterListener(handle listener.Handle) error
	UnregisterListener()
	Forward(Target, interface{}) (interface{}, error)
}

var (
	// Modules map
	sources map[v1.RuleEndpointTypeDef]SourceFactory
)

func init() {
	sources = make(map[v1.RuleEndpointTypeDef]SourceFactory)
}

// RegisterSource register module
func RegisterSource(s SourceFactory) {
	sources[s.Type()] = s
	klog.V(4).Info("source " + s.Type() + " registered")
}

// get source map
func GetSourceFactory(name v1.RuleEndpointTypeDef) (SourceFactory, bool) {
	source, exist := sources[name]
	return source, exist
}
