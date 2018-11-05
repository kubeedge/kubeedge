package record

import (
	"kubeedge/beehive/pkg/common/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type EventRecorder struct{}

func NewEventRecorder() *EventRecorder {
	return &EventRecorder{}
}

func (er *EventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	log.LOGGER.Infof("%s %s %s", eventtype, reason, message)
}

func (er *EventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	log.LOGGER.Infof(eventtype+" "+reason+" "+messageFmt, args...)
}

func (er *EventRecorder) PastEventf(object runtime.Object, timestamp metav1.Time, eventtype, reason, messageFmt string, args ...interface{}) {
}
