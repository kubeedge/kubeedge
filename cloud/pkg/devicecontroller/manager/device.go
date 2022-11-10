package manager

import (
	"reflect"

	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

// DeviceManager is a manager watch device change event
type DeviceManager struct {
	// events from watch kubernetes api server
	events chan EventWithOldObject
}

// Events return a channel, can receive all device event
func (dmm *DeviceManager) Events() chan EventWithOldObject {
	return dmm.events
}

// NewDeviceManager create DeviceManager from config
func NewDeviceManager(si cache.SharedIndexInformer) (*DeviceManager, error) {
	events := make(chan EventWithOldObject, config.Config.Buffer.DeviceEvent)
	rh := NewCommonResourceEventHandler(events, &deviceEventFilter{})
	si.AddEventHandler(rh)

	return &DeviceManager{events: events}, nil
}

var _ EventFilter = &deviceEventFilter{}

type deviceEventFilter struct{}

func (filter *deviceEventFilter) Create(obj interface{}) bool {
	return true
}

func (filter *deviceEventFilter) Delete(obj interface{}) bool {
	return true
}

func (filter *deviceEventFilter) Update(oldObj, newObj interface{}) bool {
	curDevice := newObj.(*v1alpha2.Device)
	oldDevice := oldObj.(*v1alpha2.Device)

	return isDeviceUpdated(oldDevice, curDevice)
}

// isDeviceUpdated checks if device is actually updated
func isDeviceUpdated(old *v1alpha2.Device, new *v1alpha2.Device) bool {
	// does not care fields
	old.ObjectMeta.ResourceVersion = new.ObjectMeta.ResourceVersion
	old.ObjectMeta.Generation = new.ObjectMeta.Generation

	// return true if ObjectMeta or Spec or Status changed, else false
	return !reflect.DeepEqual(old.ObjectMeta, new.ObjectMeta) ||
		!reflect.DeepEqual(old.Spec, new.Spec) ||
		!reflect.DeepEqual(old.Status, new.Status)
}
