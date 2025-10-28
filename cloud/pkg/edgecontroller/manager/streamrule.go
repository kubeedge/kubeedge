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

package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

type StreamRuleManager struct {
	events chan watch.Event
}

func (srm *StreamRuleManager) Events() chan watch.Event {
	return srm.events
}

func NewStreamRuleManager(config *v1alpha1.EdgeController, si cache.SharedIndexInformer) (*StreamRuleManager, error) {
	events := make(chan watch.Event, config.Buffer.StreamRulesEvent)
	rh := NewCommonResourceEventHandler(events, nil)
	si.AddEventHandler(rh)

	return &StreamRuleManager{events: events}, nil
}
