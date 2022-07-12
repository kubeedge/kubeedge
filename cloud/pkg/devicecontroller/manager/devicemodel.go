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

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
)

// DeviceModelManager is a manager watch DeviceModel change event
type DeviceModelManager struct {
	// events from watch kubernetes api server
	events chan watch.Event

	// DeviceModel, key is DeviceModel.Name, value is *v1alpha2.DeviceModel{}
	DeviceModel sync.Map
}

// Events return a channel, can receive all DeviceModel event
func (dmm *DeviceModelManager) Events() chan watch.Event {
	return dmm.events
}

// NewDeviceModelManager create DeviceModelManager from config
func NewDeviceModelManager(si cache.SharedIndexInformer) (*DeviceModelManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.DeviceModelEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &DeviceModelManager{events: events}, nil
}
