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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func TestNewEvents(t *testing.T) {
	assert := assert.New(t)

	s := newSend()

	events := newEvents(namespace, s)

	assert.NotNil(events)
	assert.Equal(namespace, events.namespace)
	assert.IsType(&send{}, events.send)
}

func TestEventInterface(t *testing.T) {
	assert := assert.New(t)

	s := newSend()

	e := newEvents(namespace, s)

	t.Run("Create method", func(t *testing.T) {
		evt := &corev1.Event{
			ObjectMeta: metav1.ObjectMeta{Name: "CreateEvent", Namespace: namespace},
		}
		event, err := e.Create(evt, metav1.CreateOptions{})

		assert.NoError(err)
		assert.Equal(event, evt)
	})

	t.Run("Update method", func(t *testing.T) {
		evt := &corev1.Event{
			ObjectMeta: metav1.ObjectMeta{Name: "UpdateEvent", Namespace: namespace},
		}
		event, err := e.Update(evt, metav1.UpdateOptions{})

		assert.NoError(err)
		assert.Equal(event, evt)
	})

	t.Run("Patch method", func(t *testing.T) {
		event, err := e.Patch("PatchSth", types.JSONPatchType, []byte("{\"key\": \"val\"}"), metav1.PatchOptions{})

		assert.NoError(err)
		assert.Equal(*event, corev1.Event{})
	})

	t.Run("Delete method", func(t *testing.T) {
		err := e.Delete("DeleteSth", metav1.DeleteOptions{})

		assert.NoError(err)
	})

	t.Run("Get method", func(t *testing.T) {
		event, err := e.Get("GetSth", metav1.GetOptions{})

		assert.NoError(err)
		assert.Equal(*event, corev1.Event{})
	})

	t.Run("Apply method", func(t *testing.T) {
		event, err := e.Apply(&appcorev1.EventApplyConfiguration{}, metav1.ApplyOptions{})

		assert.NoError(err)
		assert.Equal(*event, corev1.Event{})
	})
}

func TestEventExtensionInterface(t *testing.T) {
	assert := assert.New(t)
	mockSend := &mockSendInterface{}
	mockEvent := newEvents(namespace, mockSend)
	mockSend.sendFunc = func(message *model.Message) {
		assert.Equal(modules.MetaGroup, message.GetGroup())
		assert.Equal(modules.EdgedModuleName, message.GetSource())
		assert.NotEmpty(message.GetID())
		assert.Equal("test-namespace/event/test-event", message.GetResource())
	}

	t.Run("CreateWithEventNamespace", func(t *testing.T) {
		evt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{
			Name:         "test-event",
			GenerateName: "",
			Namespace:    namespace,
		}}
		result, err := mockEvent.CreateWithEventNamespace(evt)
		assert.Equal(err, nil)
		assert.Equal(result, evt)
	})

	t.Run("UpdateWithEventNamespace", func(t *testing.T) {
		evt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: namespace,
		}}
		result, err := mockEvent.UpdateWithEventNamespace(evt)
		assert.Equal(err, nil)
		assert.Equal(result, evt)
	})

	t.Run("PatchWithEventNamespace", func(t *testing.T) {
		evt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: namespace,
		}}
		result, err := mockEvent.PatchWithEventNamespace(evt, []byte("{\"key\": \"val\"}"))
		assert.Equal(err, nil)
		assert.Equal(result, evt)
	})
}
