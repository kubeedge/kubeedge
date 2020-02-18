package manager

import (
	"sync"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
)

// DeviceModelManager is a manager watch DeviceModel change event
type DeviceModelManager struct {
	// events from watch kubernetes api server
	events chan watch.Event

	// DeviceModel, key is DeviceModel.Name, value is *v1alpha1.DeviceModel{}
	DeviceModel sync.Map
}

// Events return a channel, can receive all DeviceModel event
func (dmm *DeviceModelManager) Events() chan watch.Event {
	return dmm.events
}

// NewDeviceModelManager create DeviceModelManager from config
func NewDeviceModelManager(crdClient *rest.RESTClient, namespace string) (*DeviceModelManager, error) {
	lw := cache.NewListWatchFromClient(crdClient, "devicemodels", namespace, fields.Everything())
	events := make(chan watch.Event, config.Config.Buffer.DeviceModelEvent)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1alpha1.DeviceModel{}, 0)
	si.AddEventHandler(rh)

	pm := &DeviceModelManager{events: events}

	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return pm, nil
}
