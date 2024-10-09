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

package v1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// EventBridge is a structure that handles event operations.
// FakeEvents is the whole set of Event api, and MetaClient includes a subset of FakeEvents.
type EventsBridge struct {
	fakecorev1.FakeEvents
	ns         string
	MetaClient client.CoreInterface
}

// Only XXXWithNamespace methods are actually used, the remaining methods are placeholders.
func (e *EventsBridge) Create(_ context.Context, event *corev1.Event, opts metav1.CreateOptions) (*corev1.Event, error) {
	return e.MetaClient.Events(e.ns).Create(event, opts)
}

func (e *EventsBridge) Update(_ context.Context, event *corev1.Event, opts metav1.UpdateOptions) (*corev1.Event, error) {
	return e.MetaClient.Events(e.ns).Update(event, opts)
}

func (e *EventsBridge) Patch(_ context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.Event, err error) {
	return e.MetaClient.Events(e.ns).Patch(name, pt, data, opts, subresources...)
}

func (e *EventsBridge) Delete(_ context.Context, name string, opts metav1.DeleteOptions) error {
	return e.MetaClient.Events(e.ns).Delete(name, opts)
}

func (e *EventsBridge) Get(_ context.Context, name string, opts metav1.GetOptions) (*corev1.Event, error) {
	return e.MetaClient.Events(e.ns).Get(name, opts)
}

func (e *EventsBridge) Apply(_ context.Context, event *appcorev1.EventApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.Event, err error) {
	return e.MetaClient.Events(e.ns).Apply(event, opts)
}

func (e *EventsBridge) CreateWithEventNamespace(event *corev1.Event) (*corev1.Event, error) {
	return e.MetaClient.Events(event.Namespace).CreateWithEventNamespace(event)
}

func (e *EventsBridge) UpdateWithEventNamespace(event *corev1.Event) (*corev1.Event, error) {
	return e.MetaClient.Events(event.Namespace).UpdateWithEventNamespace(event)
}

func (e *EventsBridge) PatchWithEventNamespace(event *corev1.Event, data []byte) (*corev1.Event, error) {
	return e.MetaClient.Events(event.Namespace).PatchWithEventNamespace(event, data)
}
