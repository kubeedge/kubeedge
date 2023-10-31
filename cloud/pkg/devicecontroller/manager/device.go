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

	// Device, key is device.Name, value is *v1beta1.Device{}
	// to be deprecated
	Device sync.Map

	// DeployedDevice stores the deployed devices, key is device.Name, value is *v1beta1.Device{}
	DeployedDevice sync.Map

	// UndeployedDevice stores the undeployed devices, key is device.Name, value is *v1beta1.Device{}
	UndeployedDevice sync.Map

	// NodeDeviceList stores the device list deployed on the corresponding node, key is node.Name, value is []*v1beta1.Device{}
	NodeDeviceList sync.Map
}

// Events return a channel, can receive all device event
func (dmm *DeviceManager) Events() chan watch.Event {
	return dmm.events
}

// NewDeviceManager create DeviceManager from config
func NewDeviceManager(si cache.SharedIndexInformer) (*DeviceManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.DeviceEvent)
	rh := NewCommonResourceEventHandler(events)
	_, err := si.AddEventHandler(rh)
	if err != nil {
		return nil, err
	}

	return &DeviceManager{events: events}, nil
}
