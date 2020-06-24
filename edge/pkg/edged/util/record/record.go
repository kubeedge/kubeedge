/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package record

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
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
