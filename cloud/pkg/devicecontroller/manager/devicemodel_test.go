package manager

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

func TestDeviceModelManager_Events(t *testing.T) {
	e := make(chan watch.Event, 1)
	e <- watch.Event{Type: watch.Added}
	tests := []struct {
		name   string
		events chan watch.Event
		want   chan watch.Event
	}{
		{
			name:   "base",
			events: e,
			want:   e,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dmm := &DeviceModelManager{
				events: tt.events,
			}
			if got := dmm.Events(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeviceModelManager.Events() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockInformer struct {
	cache.SharedIndexInformer
}

func (i *mockInformer) AddEventHandler(handler cache.ResourceEventHandler) {
}

func TestNewDeviceModelManager(t *testing.T) {
	dt := int32(1)
	config.Config = config.Configure{
		DeviceController: v1alpha1.DeviceController{
			Buffer: &v1alpha1.DeviceControllerBuffer{
				DeviceEvent: dt,
			},
		},
	}
	e := make(chan watch.Event, dt)
	tests := []struct {
		name    string
		si      cache.SharedIndexInformer
		want    *DeviceModelManager
		wantErr bool
	}{
		{
			name: "base",
			si:   &mockInformer{},
			want: &DeviceModelManager{
				events: e,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDeviceModelManager(tt.si)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDeviceModelManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(len(got.events), len(tt.want.events)) {
				t.Errorf("NewDeviceModelManager() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
