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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

const testEventName = "test-event"

func TestNewEvents(t *testing.T) {
	mockSend := newMockSend()

	events := newEvents(testNamespace, mockSend)

	assert.NotNil(t, events)
	assert.Equal(t, testNamespace, events.namespace)
	assert.Equal(t, mockSend, events.send)
}

func TestEvents_Create(t *testing.T) {
	testCases := []struct {
		name      string
		event     *corev1.Event
		expectErr bool
	}{
		{
			name: "Create Event Success",
			event: &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testEventName,
					Namespace: testNamespace,
				},
				Type:   corev1.EventTypeNormal,
				Reason: "TestReason",
			},
			expectErr: false,
		},
		{
			name:      "Create with nil Event",
			event:     nil,
			expectErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			eventsClient := newEvents(testNamespace, mockSend)

			result, err := eventsClient.Create(test.event, metav1.CreateOptions{})

			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.event, result)
			}
		})
	}
}

func TestEvents_Update(t *testing.T) {
	testCases := []struct {
		name      string
		event     *corev1.Event
		expectErr bool
	}{
		{
			name: "Update Event Success",
			event: &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testEventName,
					Namespace: testNamespace,
				},
				Type:   corev1.EventTypeWarning,
				Reason: "UpdatedReason",
			},
			expectErr: false,
		},
		{
			name:      "Update with nil Event",
			event:     nil,
			expectErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			eventsClient := newEvents(testNamespace, mockSend)

			result, err := eventsClient.Update(test.event, metav1.UpdateOptions{})

			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.event, result)
			}
		})
	}
}

func TestEvents_Patch(t *testing.T) {
	testCases := []struct {
		name      string
		eventName string
		patchType types.PatchType
		patchData []byte
		expectErr bool
	}{
		{
			name:      "Patch Event Success",
			eventName: testEventName,
			patchType: types.JSONPatchType,
			patchData: []byte(`[{"op":"replace","path":"/reason","value":"PatchedReason"}]`),
			expectErr: false,
		},
		{
			name:      "Patch with empty name",
			eventName: "",
			patchType: types.JSONPatchType,
			patchData: []byte(`[{"op":"replace","path":"/reason","value":"PatchedReason"}]`),
			expectErr: false,
		},
		{
			name:      "Patch with invalid patch data",
			eventName: testEventName,
			patchType: types.JSONPatchType,
			patchData: []byte(`invalid`),
			expectErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			eventsClient := newEvents(testNamespace, mockSend)

			result, err := eventsClient.Patch(test.eventName, test.patchType, test.patchData, metav1.PatchOptions{})

			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestEvents_Delete(t *testing.T) {
	testCases := []struct {
		name      string
		eventName string
		expectErr bool
	}{
		{
			name:      "Delete Event Success",
			eventName: testEventName,
			expectErr: false,
		},
		{
			name:      "Delete with empty name",
			eventName: "",
			expectErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			eventsClient := newEvents(testNamespace, mockSend)

			err := eventsClient.Delete(test.eventName, metav1.DeleteOptions{})

			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEvents_Get(t *testing.T) {
	testCases := []struct {
		name      string
		eventName string
		expectErr bool
	}{
		{
			name:      "Get Event Success",
			eventName: testEventName,
			expectErr: false,
		},
		{
			name:      "Get with empty name",
			eventName: "",
			expectErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			eventsClient := newEvents(testNamespace, mockSend)

			result, err := eventsClient.Get(test.eventName, metav1.GetOptions{})

			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, &corev1.Event{}, result)
			}
		})
	}
}

func TestEvents_Apply(t *testing.T) {
	testCases := []struct {
		name      string
		config    *appcorev1.EventApplyConfiguration
		expectErr bool
	}{
		{
			name:      "Apply Event Config Success",
			config:    &appcorev1.EventApplyConfiguration{},
			expectErr: false,
		},
		{
			name:      "Apply with nil config",
			config:    nil,
			expectErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			eventsClient := newEvents(testNamespace, mockSend)

			result, err := eventsClient.Apply(test.config, metav1.ApplyOptions{})

			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

func TestEvents_CreateWithEventNamespace(t *testing.T) {
	testCases := []struct {
		name      string
		eventName string
		event     *corev1.Event
		respFunc  func(*model.Message) (*model.Message, error)
		expectErr bool
		errMsg    string
	}{
		{
			name:      "Create Event Success",
			eventName: testEventName,
			event: &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testEventName,
					Namespace: testNamespace,
				},
				Type:    corev1.EventTypeNormal,
				Reason:  "TestReason",
				Message: "Test message",
			},
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = "OK"
				return resp, nil
			},
			expectErr: false,
		},
		{
			name:      "Create Event Network Error",
			eventName: testEventName,
			event: &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testEventName,
					Namespace: testNamespace,
				},
			},
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("network error")
			},
			expectErr: true,
			errMsg:    "send error",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendFunc = func(message *model.Message) {
				assert.Equal(t, modules.MetaGroup, message.GetGroup())
				assert.Equal(t, modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(t, message.GetID())
				assert.Equal(t, fmt.Sprintf("%s/event/%s", testNamespace, test.eventName), message.GetResource())
				assert.Equal(t, model.InsertOperation, message.GetOperation())
			}

			eventsClient := newEvents(testNamespace, mockSend)
			result, err := eventsClient.CreateWithEventNamespace(test.event)

			if test.expectErr {
				// CreateWithEventNamespace doesn't return error, it just sends
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.event, result)
			}
		})
	}
}

func TestEvents_UpdateWithEventNamespace(t *testing.T) {
	testCases := []struct {
		name  string
		event *corev1.Event
	}{
		{
			name: "Update Event Success",
			event: &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testEventName,
					Namespace: testNamespace,
				},
				Type:   corev1.EventTypeWarning,
				Reason: "UpdatedReason",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendFunc = func(message *model.Message) {
				assert.Equal(t, modules.MetaGroup, message.GetGroup())
				assert.Equal(t, modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(t, message.GetID())
				assert.Equal(t, fmt.Sprintf("%s/event/%s", testNamespace, testEventName), message.GetResource())
				assert.Equal(t, model.UpdateOperation, message.GetOperation())
			}

			eventsClient := newEvents(testNamespace, mockSend)
			result, err := eventsClient.UpdateWithEventNamespace(test.event)

			assert.NoError(t, err)
			assert.Equal(t, test.event, result)
		})
	}
}

func TestEvents_PatchWithEventNamespace(t *testing.T) {
	patchData := []byte(`[{"op":"replace","path":"/message","value":"Patched message"}]`)

	testCases := []struct {
		name      string
		event     *corev1.Event
		patchData []byte
	}{
		{
			name: "Patch Event Success",
			event: &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testEventName,
					Namespace: testNamespace,
				},
			},
			patchData: patchData,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(t, modules.MetaGroup, message.GetGroup())
				assert.Equal(t, modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(t, message.GetID())
				assert.Equal(t, fmt.Sprintf("%s/event/%s", testNamespace, testEventName), message.GetResource())
				assert.Equal(t, model.PatchOperation, message.GetOperation())

				resp := model.NewMessage(message.GetID())
				resp.Content = test.event
				return resp, nil
			}

			eventsClient := newEvents(testNamespace, mockSend)
			result, err := eventsClient.PatchWithEventNamespace(test.event, test.patchData)

			assert.NoError(t, err)
			assert.Equal(t, test.event, result)
		})
	}
}

func TestEvents_Interface(t *testing.T) {
	// Verify that events implements EventsInterface
	var _ EventsInterface = &events{}

	mockSend := newMockSend()
	eventsClient := newEvents(testNamespace, mockSend)

	var _ EventsInterface = eventsClient
	assert.NotNil(t, eventsClient)
}

func TestEvents_WithDifferentEventTypes(t *testing.T) {
	testCases := []struct {
		name      string
		eventType string
		reason    string
	}{
		{
			name:      "Normal Event",
			eventType: corev1.EventTypeNormal,
			reason:    "PodCreated",
		},
		{
			name:      "Warning Event",
			eventType: corev1.EventTypeWarning,
			reason:    "BackOff",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			event := &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testEventName,
					Namespace: testNamespace,
				},
				Type:   test.eventType,
				Reason: test.reason,
			}

			mockSend := newMockSend()
			eventsClient := newEvents(testNamespace, mockSend)

			result, err := eventsClient.Create(event, metav1.CreateOptions{})

			assert.NoError(t, err)
			assert.Equal(t, test.eventType, result.Type)
			assert.Equal(t, test.reason, result.Reason)
		})
	}
}
