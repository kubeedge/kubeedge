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

	v1 "github.com/kubeedge/kubeedge/pkg/apis/rules/v1"
)

type TargetFactory interface {
	Type() v1.RuleEndpointTypeDef
	GetTarget(ep *v1.RuleEndpoint, targetResource map[string]string) Target
}

type Target interface {
	Name() string
	GoToTarget(data map[string]interface{}, stop chan struct{}) (interface{}, error)
}

var (
	// Modules map
	targets map[v1.RuleEndpointTypeDef]TargetFactory
)

func init() {
	targets = make(map[v1.RuleEndpointTypeDef]TargetFactory)
}

// RegisterSource register module
func RegisterTarget(t TargetFactory) {
	targets[t.Type()] = t
	klog.V(4).Info("target " + t.Type() + " registered")
}

// get source map
func GetTargetFactory(name v1.RuleEndpointTypeDef) (TargetFactory, bool) {
	target, exist := targets[name]
	return target, exist
}
