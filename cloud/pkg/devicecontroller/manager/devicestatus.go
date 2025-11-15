package manager

import (
	"sync"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
)

// DeviceStatusManager is a manager watch DeviceStatus change event
type DeviceStatusManager struct {
	// events from watch kubernetes api server
	events chan watch.Event

	// DeviceStatus, key is DeviceStatus.Namespace+"/"+DeviceStatus.Name, value is *v1beta1.DeviceStatus{}
	DeviceStatus sync.Map
}

// Events return a channel, can receive all DeviceStatus event
func (dsm *DeviceStatusManager) Events() chan watch.Event {
	return dsm.events
}

// NewDeviceStatusManager create DeviceStatusManager from config
func NewDeviceStatusManager(si cache.SharedIndexInformer) (*DeviceStatusManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.DeviceStatusEvent)
	rh := NewCommonResourceEventHandler(events)
	_, err := si.AddEventHandler(rh)
	if err != nil {
		return nil, err
	}

	return &DeviceStatusManager{events: events}, nil
}
