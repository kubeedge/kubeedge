package manager

import (
	"reflect"

	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
)

// DeviceModelManager is a manager watch DeviceModel change event
type DeviceModelManager struct {
	// events from watch kubernetes api server
	events chan EventWithOldObject
}

// Events return a channel, can receive all DeviceModel event
func (dmm *DeviceModelManager) Events() chan EventWithOldObject {
	return dmm.events
}

// NewDeviceModelManager create DeviceModelManager from config
func NewDeviceModelManager(si cache.SharedIndexInformer) (*DeviceModelManager, error) {
	events := make(chan EventWithOldObject, config.Config.Buffer.DeviceModelEvent)
	rh := NewCommonResourceEventHandler(events, &deviceModelEventFilter{})
	si.AddEventHandler(rh)

	return &DeviceModelManager{events: events}, nil
}

var _ EventFilter = &deviceModelEventFilter{}

type deviceModelEventFilter struct{}

func (filter *deviceModelEventFilter) Create(obj interface{}) bool {
	return true
}

func (filter *deviceModelEventFilter) Delete(obj interface{}) bool {
	return true
}

func (filter *deviceModelEventFilter) Update(oldObj, newObj interface{}) bool {
	curModel := newObj.(*v1alpha2.DeviceModel)
	oldModel := oldObj.(*v1alpha2.DeviceModel)

	return isDeviceModelUpdated(oldModel, curModel)
}

// isDeviceModelUpdated is function to check if deviceModel is actually updated
func isDeviceModelUpdated(oldTwin *v1alpha2.DeviceModel, newTwin *v1alpha2.DeviceModel) bool {
	// does not care fields
	oldTwin.ObjectMeta.ResourceVersion = newTwin.ObjectMeta.ResourceVersion
	oldTwin.ObjectMeta.Generation = newTwin.ObjectMeta.Generation

	// return true if ObjectMeta or Spec or Status changed, else false
	return !reflect.DeepEqual(oldTwin.ObjectMeta, newTwin.ObjectMeta) || !reflect.DeepEqual(oldTwin.Spec, newTwin.Spec)
}
