package record

import (
	"github.com/kubeedge/beehive/pkg/common/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//EventRecorder is object type to record events
type EventRecorder struct{}

//NewEventRecorder creates and returns event recorder object
func NewEventRecorder() *EventRecorder {
	return &EventRecorder{}
}

//Event logs info of event
func (er *EventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	log.LOGGER.Infof("%s %s %s", eventtype, reason, message)
}

//Eventf logs info of event
func (er *EventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	log.LOGGER.Infof(eventtype+" "+reason+" "+messageFmt, args...)
}

//PastEventf logs past events info
func (er *EventRecorder) PastEventf(object runtime.Object, timestamp metav1.Time, eventtype, reason, messageFmt string, args ...interface{}) {
}
