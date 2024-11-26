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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func TestNewPods(t *testing.T) {
	assert := assert.New(t)

	send := newSend()

	pods := newPods(namespace, send)
	assert.NotNil(pods)
	assert.Equal(namespace, pods.namespace)
	assert.Equal(send, pods.send)
}

func TestPods_Delete(t *testing.T) {
	assert := assert.New(t)

	podName := "test-pod"
	deleteOptions := metav1.DeleteOptions{}

	testCases := []struct {
		name      string
		respFunc  func(*model.Message) (*model.Message, error)
		expectErr bool
	}{
		{
			name: "Delete Pod Success",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = "OK"
				return resp, nil
			},
			expectErr: false,
		},
		{
			name: "Delete Pod Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("test error")
			},
			expectErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(modules.MetaGroup, message.GetGroup())
				assert.Equal(modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(message.GetID())
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypePod, podName), message.GetResource())
				assert.Equal(model.DeleteOperation, message.GetOperation())

				return test.respFunc(message)
			}

			podsClient := newPods(namespace, mockSend)

			err := podsClient.Delete(podName, deleteOptions)

			if test.expectErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestPods_Get(t *testing.T) {
	assert := assert.New(t)

	podName := "test-pod"
	expectedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	testCases := []struct {
		name      string
		respFunc  func(*model.Message) (*model.Message, error)
		stdResult *corev1.Pod
		expectErr bool
	}{
		{
			name: "Get Pod Success",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				podJSON, _ := json.Marshal(expectedPod)
				resp.Content = []string{string(podJSON)}
				return resp, nil
			},
			stdResult: expectedPod,
			expectErr: false,
		},
		{
			name: "Get Pod Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("test error")
			},
			stdResult: nil,
			expectErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(modules.MetaGroup, message.GetGroup())
				assert.Equal(modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(message.GetID())
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypePod, podName), message.GetResource())
				assert.Equal(model.QueryOperation, message.GetOperation())

				return test.respFunc(message)
			}

			podsClient := newPods(namespace, mockSend)

			pod, err := podsClient.Get(podName)

			if test.expectErr {
				assert.Error(err)
				assert.Nil(pod)
			} else {
				assert.NoError(err)
				assert.Equal(test.stdResult, pod)
			}
		})
	}
}

func TestHandlePodFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name        string
		content     []byte
		expectedPod *corev1.Pod
		expectedErr bool
	}{
		{
			name:    "Valid Pod",
			content: []byte(`["{\"metadata\":{\"name\":\"test-pod\",\"namespace\":\"default\"},\"spec\":{\"containers\":[{\"name\":\"test-container\",\"image\":\"test-image\"}]}}"]`),
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
			expectedErr: false,
		},
		{
			name:        "Invalid JSON",
			content:     []byte(`["invalid json"]`),
			expectedPod: nil,
			expectedErr: true,
		},
		{
			name:        "Empty list",
			content:     []byte(`[]`),
			expectedPod: nil,
			expectedErr: true,
		},
		{
			name:        "Multiple Pods",
			content:     []byte(`["{}", "{}"]`),
			expectedPod: nil,
			expectedErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			pod, err := handlePodFromMetaDB("test-pod", test.content)

			if test.expectedErr {
				assert.Error(err)
				assert.Nil(pod)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedPod.ObjectMeta.Name, pod.ObjectMeta.Name)
				assert.Equal(test.expectedPod.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
				assert.Equal(test.expectedPod.Spec.Containers[0].Name, pod.Spec.Containers[0].Name)
				assert.Equal(test.expectedPod.Spec.Containers[0].Image, pod.Spec.Containers[0].Image)
			}
		})
	}
}
