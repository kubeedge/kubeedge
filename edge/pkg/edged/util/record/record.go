package record

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

//EventRecorder is object type to record events
type EventRecorder struct{}

//NewEventRecorder creates and returns event recorder object
func NewEventRecorder() *EventRecorder {
	return &EventRecorder{}
}

//Event logs info of event
func (er *EventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	klog.Infof("%s %s %s", eventtype, reason, message)
}

//Eventf logs info of event
func (er *EventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	klog.Infof(eventtype+" "+reason+" "+messageFmt, args...)
}

//PastEventf logs past events info
func (er *EventRecorder) PastEventf(object runtime.Object, timestamp metav1.Time, eventtype, reason, messageFmt string, args ...interface{}) {
}

// AnnotatedEventf is just like eventf, but with annotations attached
func (er *EventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
}
