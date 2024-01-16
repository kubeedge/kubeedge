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

	// DeviceModel, key is DeviceModel.Namespace+"/"+deviceModel.Name, value is *v1beta1.DeviceModel{}
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
	_, err := si.AddEventHandler(rh)
	if err != nil {
		return nil, err
	}

	return &DeviceModelManager{events: events}, nil
}
