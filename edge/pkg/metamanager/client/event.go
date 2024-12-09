/*
Copyright 2024 The KubeEdge Authors.

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

package client

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

type EventsGetter interface {
	Events(namespace string) EventsInterface
}

type EventsInterface interface {
	Create(*corev1.Event, metav1.CreateOptions) (*corev1.Event, error)
	Update(*corev1.Event, metav1.UpdateOptions) (*corev1.Event, error)
	Patch(string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*corev1.Event, error)
	Delete(string, metav1.DeleteOptions) error
	Get(string, metav1.GetOptions) (*corev1.Event, error)
	Apply(*appcorev1.EventApplyConfiguration, metav1.ApplyOptions) (result *corev1.Event, err error)

	CreateWithEventNamespace(*corev1.Event) (*corev1.Event, error)
	UpdateWithEventNamespace(*corev1.Event) (*corev1.Event, error)
	PatchWithEventNamespace(*corev1.Event, []byte) (*corev1.Event, error)
}

type events struct {
	send      SendInterface
	namespace string
}

func newEvents(namespace string, s SendInterface) *events {
	return &events{
		send:      s,
		namespace: namespace,
	}
}

func (e *events) Create(event *corev1.Event, opts metav1.CreateOptions) (*corev1.Event, error) {
	klog.Errorf("failed to create event %v with option %v", event, opts)
	return event, nil
}

func (e *events) Update(event *corev1.Event, opts metav1.UpdateOptions) (*corev1.Event, error) {
	klog.Errorf("failed to update event %v with option %v", event, opts)
	return event, nil
}

func (e *events) Patch(name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*corev1.Event, error) {
	klog.Errorf("failed to patch event with eventName %v, type %v, patchData %v, option %v and subresources %v", name, pt, string(data), opts, subresources)
	return &corev1.Event{}, nil
}

func (e *events) Delete(name string, opts metav1.DeleteOptions) error {
	klog.Errorf("failed to delete event with eventName %v and option %v", name, opts)
	return nil
}

func (e *events) Get(name string, opts metav1.GetOptions) (*corev1.Event, error) {
	klog.Errorf("failed to get event with eventName %v and option %v", name, opts)
	return &corev1.Event{}, nil
}

func (e *events) Apply(event *appcorev1.EventApplyConfiguration, opts metav1.ApplyOptions) (*corev1.Event, error) {
	klog.Errorf("failed to apply event %v with option %v", event, opts)
	return &corev1.Event{}, nil
}

func (e *events) CreateWithEventNamespace(event *corev1.Event) (*corev1.Event, error) {
	resource := fmt.Sprintf("%s/%s/%s", event.Namespace, model.ResourceTypeEvent, event.Name)
	eventMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, event)
	e.send.Send(eventMsg)
	return event, nil
}

func (e *events) UpdateWithEventNamespace(event *corev1.Event) (*corev1.Event, error) {
	resource := fmt.Sprintf("%s/%s/%s", event.Namespace, model.ResourceTypeEvent, event.Name)
	eventMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.UpdateOperation, event)
	e.send.Send(eventMsg)
	return event, nil
}

func (e *events) PatchWithEventNamespace(event *corev1.Event, data []byte) (*corev1.Event, error) {
	msgData := edgeapi.EventPatchInfo{
		Event: event,
		Data:  string(data),
	}
	resource := fmt.Sprintf("%s/%s/%s", event.Namespace, model.ResourceTypeEvent, event.Name)
	eventMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.PatchOperation, msgData)
	e.send.Send(eventMsg)
	return event, nil
}
