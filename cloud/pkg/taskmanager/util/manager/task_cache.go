/*
Copyright 2023 The KubeEdge Authors.

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

	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/config"
)

// TaskCache is a manager watch CRD change event
type TaskCache struct {
	// events from watch kubernetes api server
	events chan watch.Event

	// CacheMap, key is NodeUpgradeJob.Name, value is *v1alpha1.NodeUpgradeJob{}
	CacheMap sync.Map
}

// Events return a channel, can receive all NodeUpgradeJob event
func (dmm *TaskCache) Events() chan watch.Event {
	return dmm.events
}

// NewTaskCache create TaskCache from config
func NewTaskCache(si cache.SharedIndexInformer) (*TaskCache, error) {
	events := make(chan watch.Event, config.Config.Buffer.TaskEvent)
	rh := NewCommonResourceEventHandler(events)
	_, err := si.AddEventHandler(rh)
	if err != nil {
		return nil, err
	}

	return &TaskCache{events: events}, nil
}
