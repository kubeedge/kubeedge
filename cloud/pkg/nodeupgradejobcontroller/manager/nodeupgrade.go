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

package manager

import (
	"sync"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/nodeupgradejobcontroller/config"
)

// NodeUpgradeJobManager is a manager watch NodeUpgradeJob change event
type NodeUpgradeJobManager struct {
	// events from watch kubernetes api server
	events chan watch.Event

	// UpgradeMap, key is NodeUpgradeJob.Name, value is *v1alpha1.NodeUpgradeJob{}
	UpgradeMap sync.Map
}

// Events return a channel, can receive all NodeUpgradeJob event
func (dmm *NodeUpgradeJobManager) Events() chan watch.Event {
	return dmm.events
}

// NewNodeUpgradeJobManager create NodeUpgradeJobManager from config
func NewNodeUpgradeJobManager(si cache.SharedIndexInformer) (*NodeUpgradeJobManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.NodeUpgradeJobEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &NodeUpgradeJobManager{events: events}, nil
}
