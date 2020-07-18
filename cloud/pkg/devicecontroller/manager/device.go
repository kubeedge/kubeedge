package manager

import (
	"sync"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
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
func NewDeviceManager(crdClient *rest.RESTClient, namespace string) (*DeviceManager, error) {
	lw := cache.NewListWatchFromClient(crdClient, "devices", namespace, fields.Everything())
	events := make(chan watch.Event, config.Config.Buffer.DeviceEvent)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1alpha2.Device{}, 0)
	si.AddEventHandler(rh)

	pm := &DeviceManager{events: events}

	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return pm, nil
}
