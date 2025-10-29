/*
Copyright 2025 The KubeEdge Authors.

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

	"github.com/kubeedge/api/apis/streamrules/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/listener"
)

type TargetFactory interface {
	Type() v1alpha1.ProtocolType
	GetTarget(ep *v1alpha1.StreamRuleEndpoint, targetResource map[string]string) Target
}

type Target interface {
	Name() string
	RegisterListener(handle listener.Handle) error
	UnregisterListener()
	SendMsg(interface{}) (interface{}, error)
}

type Targets []Target

// Modules map
var targets map[v1alpha1.ProtocolType]TargetFactory

func init() {
	targets = make(map[v1alpha1.ProtocolType]TargetFactory)
}

// RegisterTarget register module
func RegisterTarget(t TargetFactory) {
	targets[t.Type()] = t
	klog.Info("target " + string(t.Type()) + " registered")
}

// get targets map
func GetTargetFactory(name v1alpha1.ProtocolType) (TargetFactory, bool) {
	target, exist := targets[name]
	return target, exist
}
