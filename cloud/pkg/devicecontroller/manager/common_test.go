package manager

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeedge/api/apis/devices/v1beta1"
)

func TestCommonResourceEventHandlerSetMetaType(t *testing.T) {
	events := make(chan watch.Event, 1)
	handler := NewCommonResourceEventHandler(events)
	device := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-device",
			Namespace: "default",
		},
	}

	handler.OnAdd(device, false)

	select {
	case e := <-events:
		got, ok := e.Object.(*v1beta1.Device)
		if !ok {
			t.Fatalf("unexpected event object type %T", e.Object)
		}
		if got.GetObjectKind().GroupVersionKind().Kind != "Device" {
			t.Fatalf("unexpected device kind %q", got.GetObjectKind().GroupVersionKind().Kind)
		}
		if got.GetObjectKind().GroupVersionKind().GroupVersion().String() != v1beta1.SchemeGroupVersion.String() {
			t.Fatalf("unexpected device apiVersion %q", got.GetObjectKind().GroupVersionKind().GroupVersion().String())
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}
