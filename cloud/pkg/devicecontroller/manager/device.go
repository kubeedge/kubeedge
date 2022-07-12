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

// DeviceManager is a manager watch device change event
type DeviceManager struct {
	// events from watch kubernetes api server
	events chan watch.Event

	// Device, key is device.Name, value is *v1alpha2.Device{}
	Device sync.Map
}

// Events return a channel, can receive all device event
func (dmm *DeviceManager) Events() chan watch.Event {
	return dmm.events
}

// NewDeviceManager create DeviceManager from config
func NewDeviceManager(si cache.SharedIndexInformer) (*DeviceManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.DeviceEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &DeviceManager{events: events}, nil
}
