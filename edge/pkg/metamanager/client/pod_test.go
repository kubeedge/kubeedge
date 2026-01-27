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
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

const testPodName = "test-pod"

func TestNewPods(t *testing.T) {
	send := newMockSend()

	pods := newPods(testNamespace, send)

	assert.NotNil(t, pods)
	assert.Equal(t, testNamespace, pods.namespace)
	assert.Equal(t, send, pods.send)
}

func TestPods_Create(t *testing.T) {
	testCases := []struct {
		name      string
		pod       *corev1.Pod
		expectErr bool
	}{
		{
			name: "Create Pod Success",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testPodName,
					Namespace: testNamespace,
				},
			},
			expectErr: false,
		},
		{
			name:      "Create with nil Pod",
			pod:       nil,
			expectErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			podsClient := newPods(testNamespace, mockSend)

			result, err := podsClient.Create(test.pod)

			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Nil(t, result)
			}
		})
	}
}

func TestPods_Update(t *testing.T) {
	testCases := []struct {
		name      string
		pod       *corev1.Pod
		expectErr bool
	}{
		{
			name: "Update Pod Success",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testPodName,
					Namespace: testNamespace,
				},
			},
			expectErr: false,
		},
		{
			name:      "Update with nil Pod",
			pod:       nil,
			expectErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			podsClient := newPods(testNamespace, mockSend)

			err := podsClient.Update(test.pod)

			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPods_Delete(t *testing.T) {
	podName := testPodName
	deleteOptions := metav1.DeleteOptions{}

	testCases := []struct {
		name      string
		podName   string
		respFunc  func(*model.Message) (*model.Message, error)
		expectErr bool
		errMsg    string
	}{
		{
			name:    "Delete Pod Success",
			podName: podName,
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = "OK"
				return resp, nil
			},
			expectErr: false,
		},
		{
			name:    "Delete Pod Network Error",
			podName: podName,
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("network error")
			},
			expectErr: true,
			errMsg:    "delete pod from metaManager failed",
		},
		{
			name:    "Delete Pod with Empty Name",
			podName: "",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("invalid resource")
			},
			expectErr: true,
		},
		{
			name:    "Delete Pod Message Parse Error",
			podName: podName,
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = fmt.Errorf("pod not found")
				return resp, nil
			},
			expectErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(t, modules.MetaGroup, message.GetGroup())
				assert.Equal(t, modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(t, message.GetID())
				assert.Equal(t, fmt.Sprintf("%s/%s/%s", testNamespace, model.ResourceTypePod, test.podName),
					message.GetResource())
				assert.Equal(t, model.DeleteOperation, message.GetOperation())

				return test.respFunc(message)
			}

			podsClient := newPods(testNamespace, mockSend)
			err := podsClient.Delete(test.podName, deleteOptions)

			if test.expectErr {
				assert.Error(t, err)
				if test.errMsg != "" {
					assert.Contains(t, err.Error(), test.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPods_Get(t *testing.T) {
	expectedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespace,
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
		podName   string
		respFunc  func(*model.Message) (*model.Message, error)
		stdResult *corev1.Pod
		expectErr bool
		errMsg    string
	}{
		{
			name:    "Get Pod Success",
			podName: testPodName,
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				podJSON, _ := json.Marshal(expectedPod)
				resp.Content = string(podJSON)
				return resp, nil
			},
			stdResult: expectedPod,
			expectErr: false,
		},
		{
			name:    "Get Pod Network Error",
			podName: testPodName,
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("connection refused")
			},
			stdResult: nil,
			expectErr: true,
			errMsg:    "get pod from metaManager failed",
		},
		{
			name:    "Get Pod with Empty Name",
			podName: "",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("invalid resource")
			},
			stdResult: nil,
			expectErr: true,
		},
		{
			name:    "Get Pod Parse Error",
			podName: testPodName,
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = fmt.Errorf("pod not found")
				return resp, nil
			},
			stdResult: nil,
			expectErr: true,
		},
		{
			name:    "Get Pod Invalid JSON",
			podName: testPodName,
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = "invalid json"
				return resp, nil
			},
			stdResult: nil,
			expectErr: true,
			errMsg:    "parse message to pod failed",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(t, modules.MetaGroup, message.GetGroup())
				assert.Equal(t, modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(t, message.GetID())
				assert.Equal(t, fmt.Sprintf("%s/%s/%s", testNamespace, model.ResourceTypePod, test.podName),
					message.GetResource())
				assert.Equal(t, model.QueryOperation, message.GetOperation())

				return test.respFunc(message)
			}

			podsClient := newPods(testNamespace, mockSend)
			pod, err := podsClient.Get(test.podName)

			if test.expectErr {
				assert.Error(t, err)
				assert.Nil(t, pod)
				if test.errMsg != "" {
					assert.Contains(t, err.Error(), test.errMsg)
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, pod)
				assert.Equal(t, test.stdResult.ObjectMeta.Name, pod.ObjectMeta.Name)
				assert.Equal(t, test.stdResult.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
			}
		})
	}
}

func TestHandlePodFromMetaDB(t *testing.T) {
	validPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
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
	}

	testCases := []struct {
		name        string
		podName     string
		content     []byte
		expectedPod *corev1.Pod
		expectErr   bool
		errMsg      string
	}{
		{
			name:    "Valid Pod",
			podName: testPodName,
			content: func() []byte {
				podJSON, _ := json.Marshal(validPod)
				return []byte(`[` + string(podJSON) + `]`)
			}(),
			expectedPod: validPod,
			expectErr:   false,
		},
		{
			name:        "Invalid JSON",
			podName:     testPodName,
			content:     []byte(`["invalid json"]`),
			expectedPod: nil,
			expectErr:   true,
			errMsg:      "unmarshal message to pod from db failed",
		},
		{
			name:        "Empty list",
			podName:     testPodName,
			content:     []byte(`[]`),
			expectedPod: nil,
			expectErr:   true,
			errMsg:      "pod length from meta db is 0",
		},
		{
			name:        "Multiple Pods",
			podName:     testPodName,
			content:     []byte(`["{}", "{}"]`),
			expectedPod: nil,
			expectErr:   true,
			errMsg:      "pod length from meta db is 2",
		},
		{
			name:        "Malformed JSON Array",
			podName:     testPodName,
			content:     []byte(`{}`),
			expectedPod: nil,
			expectErr:   true,
			errMsg:      "unmarshal message to pod from db failed",
		},
		{
			name:        "Empty Content",
			podName:     testPodName,
			content:     []byte(``),
			expectedPod: nil,
			expectErr:   true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			pod, err := handlePodFromMetaDB(test.podName, test.content)

			if test.expectErr {
				assert.Error(t, err)
				assert.Nil(t, pod)
				if test.errMsg != "" {
					assert.Contains(t, err.Error(), test.errMsg)
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, pod)
				assert.Equal(t, test.expectedPod.ObjectMeta.Name, pod.ObjectMeta.Name)
				assert.Equal(t, test.expectedPod.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
				assert.Equal(t, len(test.expectedPod.Spec.Containers),
					len(pod.Spec.Containers))
				if len(pod.Spec.Containers) > 0 {
					assert.Equal(t, test.expectedPod.Spec.Containers[0].Name,
						pod.Spec.Containers[0].Name)
					assert.Equal(t, test.expectedPod.Spec.Containers[0].Image,
						pod.Spec.Containers[0].Image)
				}
			}
		})
	}
}

func TestPods_Interface(t *testing.T) {
	// Verify that pods implements PodsInterface
	var _ PodsInterface = &pods{}

	mockSend := newMockSend()
	podsClient := newPods(testNamespace, mockSend)

	var _ PodsInterface = podsClient
	assert.NotNil(t, podsClient)
}

func TestHandlePodFromMetaDB_WithComplexPod(t *testing.T) {
	podName := "complex-pod"
	complexPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: "kube-system",
			Labels: map[string]string{
				"app": "test",
				"env": "prod",
			},
			Annotations: map[string]string{
				"description": "test pod",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "container1",
					Image: "image:v1.0",
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}

	podJSON, err := json.Marshal(complexPod)
	require.NoError(t, err)

	content := []byte(`[` + string(podJSON) + `]`)
	pod, err := handlePodFromMetaDB(podName, content)

	assert.NoError(t, err)
	assert.NotNil(t, pod)
	assert.Equal(t, "kube-system", pod.ObjectMeta.Namespace)
	assert.Equal(t, 2, len(pod.ObjectMeta.Labels))
	assert.Equal(t, "test", pod.ObjectMeta.Labels["app"])
	assert.Equal(t, "prod", pod.ObjectMeta.Labels["env"])
	assert.Equal(t, 1, len(pod.Spec.Containers))
	assert.Equal(t, "container1", pod.Spec.Containers[0].Name)
	assert.Equal(t, "image:v1.0", pod.Spec.Containers[0].Image)
	assert.Equal(t, int32(8080), pod.Spec.Containers[0].Ports[0].ContainerPort)
	assert.Equal(t, corev1.RestartPolicyAlways, pod.Spec.RestartPolicy)
}
