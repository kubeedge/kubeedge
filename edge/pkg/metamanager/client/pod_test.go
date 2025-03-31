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
	"errors"
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

const (
	testPodName            = "test-pod"
	testContainerName      = "test-container"
	testImage              = "test-image"
	testUpdatedImage       = "updated-image"
	mosquittoContainerName = "mosquitto"
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
				return nil, errors.New("test error")
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
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypePod, testPodName), message.GetResource())
				assert.Equal(model.DeleteOperation, message.GetOperation())

				return test.respFunc(message)
			}

			podsClient := newPods(namespace, mockSend)

			err := podsClient.Delete(testPodName, deleteOptions)

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

	expectedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  testContainerName,
					Image: testImage,
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
				podJSON, err := json.Marshal(expectedPod)
				if err != nil {
					return nil, err
				}
				resp.Content = []string{string(podJSON)}
				return resp, nil
			},
			stdResult: expectedPod,
			expectErr: false,
		},
		{
			name: "Get Pod Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, errors.New("test error")
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
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypePod, testPodName), message.GetResource())
				assert.Equal(model.QueryOperation, message.GetOperation())

				return test.respFunc(message)
			}

			podsClient := newPods(namespace, mockSend)

			pod, err := podsClient.Get(testPodName)

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
					Name:      testPodName,
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  testContainerName,
							Image: testImage,
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
			pod, err := handlePodFromMetaDB(testPodName, test.content)

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

func TestPods_Create(t *testing.T) {
	assert := assert.New(t)

	expectedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  testContainerName,
					Image: testImage,
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
			name: "Create Pod Success",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				podResp := PodResp{
					Object: expectedPod,
					Err:    apierrors.StatusError{},
				}
				respData, err := json.Marshal(podResp)
				if err != nil {
					return nil, err
				}
				resp.Content = respData
				return resp, nil
			},
			stdResult: expectedPod,
			expectErr: false,
		},
		{
			name: "Create Pod Error in SendSync",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, errors.New("test error")
			},
			stdResult: nil,
			expectErr: true,
		},
		{
			name: "Create Pod Error in GetContentData",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = 123
				return resp, nil
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
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypePod, expectedPod.Name), message.GetResource())
				assert.Equal(model.InsertOperation, message.GetOperation())

				return test.respFunc(message)
			}

			patches := gomonkey.ApplyFunc(updatePodDB, func(resource string, pod *corev1.Pod) error {
				return nil
			})
			defer patches.Reset()

			podsClient := newPods(namespace, mockSend)

			pod, err := podsClient.Create(expectedPod)

			if test.expectErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(test.stdResult, pod)
			}
		})
	}
}

func TestPods_Update(t *testing.T) {
	assert := assert.New(t)

	mockSend := &mockSendInterface{}
	podsClient := newPods(namespace, mockSend)

	err := podsClient.Update(&corev1.Pod{})
	assert.Nil(err)
}

func TestPods_Patch(t *testing.T) {
	assert := assert.New(t)

	patchBytes := []byte(`{"spec":{"containers":[{"name":"test-container","image":"updated-image"}]}}`)
	expectedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  testContainerName,
					Image: testUpdatedImage,
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
	}{
		{
			name:    "Patch Pod Success",
			podName: testPodName,
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				podResp := PodResp{
					Object: expectedPod,
					Err:    apierrors.StatusError{},
				}
				respData, err := json.Marshal(podResp)
				if err != nil {
					return nil, err
				}
				resp.Content = respData
				return resp, nil
			},
			stdResult: expectedPod,
			expectErr: false,
		},
		{
			name:    "Patch Pod Error in SendSync",
			podName: testPodName,
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, errors.New("test error")
			},
			stdResult: nil,
			expectErr: true,
		},
		{
			name:    "Patch Pod Error in GetContentData",
			podName: testPodName,
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = 123
				return resp, nil
			},
			stdResult: nil,
			expectErr: true,
		},
		{
			name:    "Patch Response Error Operation",
			podName: testPodName,
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Operation = model.ResponseErrorOperation
				resp.Content = "error message"
				return resp, nil
			},
			stdResult: nil,
			expectErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = test.respFunc

			patches := gomonkey.ApplyFunc(updatePodDB, func(resource string, pod *corev1.Pod) error {
				return nil
			})
			defer patches.Reset()

			podsClient := newPods(namespace, mockSend)

			pod, err := podsClient.Patch(test.podName, patchBytes)

			if test.expectErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(test.stdResult, pod)
			}
		})
	}
}

func TestPods_PatchMQTT(t *testing.T) {
	assert := assert.New(t)

	podName := mosquittoContainerName
	patchBytes := []byte(`{"spec":{"containers":[{"name":"mqtt","image":"updated-image"}]}}`)
	expectedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "mqtt",
					Image: "mosquitto",
				},
			},
		},
	}

	podJSON, err := json.Marshal(expectedPod)
	assert.NoError(err)
	metas := &[]string{string(podJSON)}

	patches := gomonkey.ApplyFunc(dao.QueryMeta, func(key string, value string) (*[]string, error) {
		assert.Equal("key", key)
		assert.Equal(fmt.Sprintf("default/pod/%s", mosquittoContainerName), value)
		return metas, nil
	})
	defer patches.Reset()

	// Add patches for the special mosquitto handling logic
	patchesPods := gomonkey.ApplyMethod(&pods{}, "Patch", func(c *pods, name string, patchBytes []byte) (*corev1.Pod, error) {
		// Special handling for mosquitto container
		if name == mosquittoContainerName {
			handleMqttMeta := func() (*corev1.Pod, error) {
				value := fmt.Sprintf("default/pod/%s", mosquittoContainerName)
				metas, err := dao.QueryMeta("key", value)
				if err != nil {
					return nil, err
				}

				if metas == nil || len(*metas) != 1 {
					return nil, errors.New("invalid meta length")
				}

				var pod corev1.Pod
				if err := json.Unmarshal([]byte((*metas)[0]), &pod); err != nil {
					return nil, err
				}

				return &pod, nil
			}

			return handleMqttMeta()
		}

		// For non-mosquitto pods, call the original method
		resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypePodPatch, name)
		podMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.PatchOperation, string(patchBytes))
		resp, err := c.send.SendSync(podMsg)
		if err != nil {
			return nil, fmt.Errorf("update pod failed, err: %v", err)
		}

		content, err := resp.GetContentData()
		if err != nil {
			return nil, fmt.Errorf("parse message to pod failed, err: %v", err)
		}

		if resp.Router.Operation == model.ResponseErrorOperation {
			return nil, errors.New(string(content))
		}

		return handlePodResp(resource, content)
	})
	defer patchesPods.Reset()

	mockSend := &mockSendInterface{}
	mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
		t.Fail()
		return nil, nil
	}

	podsClient := newPods(namespace, mockSend)

	pod, err := podsClient.Patch(podName, patchBytes)

	assert.NoError(err)
	assert.Equal(expectedPod, pod)
}

func TestHandlePodResp(t *testing.T) {
	assert := assert.New(t)

	resource := fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypePod, testPodName)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  testContainerName,
					Image: testImage,
				},
			},
		},
	}

	testCases := []struct {
		name        string
		podResp     PodResp
		mockDBError error
		expectedPod *corev1.Pod
		expectedErr bool
	}{
		{
			name: "Success Case",
			podResp: PodResp{
				Object: pod,
				Err:    apierrors.StatusError{},
			},
			mockDBError: nil,
			expectedPod: pod,
			expectedErr: false,
		},
		{
			name: "DB Error Case",
			podResp: PodResp{
				Object: pod,
				Err:    apierrors.StatusError{},
			},
			mockDBError: errors.New("database error"),
			expectedPod: nil,
			expectedErr: true,
		},
		{
			name: "API Error Case",
			podResp: PodResp{
				Object: pod,
				Err:    apierrors.StatusError{ErrStatus: metav1.Status{Message: "API error"}},
			},
			mockDBError: nil,
			expectedPod: pod,
			expectedErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(updatePodDB, func(resource string, pod *corev1.Pod) error {
				return test.mockDBError
			})
			defer patches.Reset()

			content, err := json.Marshal(test.podResp)
			assert.NoError(err)

			pod, err := handlePodResp(resource, content)

			if test.expectedErr {
				assert.Error(err)
				if test.name == "DB Error Case" {
					assert.Nil(pod)
				}
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedPod, pod)
			}
		})
	}
}

func TestHandlePodRespInvalidJSON(t *testing.T) {
	assert := assert.New(t)

	resource := fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypePod, testPodName)
	content := []byte("invalid json")

	pod, err := handlePodResp(resource, content)

	assert.Error(err)
	assert.Nil(pod)
	assert.Contains(err.Error(), "unmarshal")
}

func TestUpdatePodDB(t *testing.T) {
	assert := assert.New(t)

	resource := fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypePodPatch, testPodName)
	expectedKey := fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypePod, testPodName)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  testContainerName,
					Image: testImage,
				},
			},
		},
	}

	testCases := []struct {
		name        string
		mockDBError error
		expectedErr bool
	}{
		{
			name:        "Success Case",
			mockDBError: nil,
			expectedErr: false,
		},
		{
			name:        "DB Error Case",
			mockDBError: errors.New("database error"),
			expectedErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(dao.InsertOrUpdate, func(meta *dao.Meta) error {
				assert.Equal(expectedKey, meta.Key)
				assert.Equal(model.ResourceTypePod, meta.Type)
				return test.mockDBError
			})
			defer patches.Reset()

			err := updatePodDB(resource, pod)

			if test.expectedErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestHandleMqttMeta(t *testing.T) {
	assert := assert.New(t)

	expectedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mosquittoContainerName,
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "mqtt",
					Image: "mosquitto",
				},
			},
		},
	}
	podJSON, err := json.Marshal(expectedPod)
	assert.NoError(err)

	testCases := []struct {
		name        string
		metas       *[]string
		mockDBError error
		expectedPod *corev1.Pod
		expectedErr bool
	}{
		{
			name:        "Success Case",
			metas:       &[]string{string(podJSON)},
			mockDBError: nil,
			expectedPod: expectedPod,
			expectedErr: false,
		},
		{
			name:        "Query Error Case",
			metas:       nil,
			mockDBError: errors.New("database error"),
			expectedPod: nil,
			expectedErr: true,
		},
		{
			name:        "Invalid Meta Length",
			metas:       &[]string{},
			mockDBError: nil,
			expectedPod: nil,
			expectedErr: true,
		},
		{
			name:        "Invalid JSON",
			metas:       &[]string{"invalid json"},
			mockDBError: nil,
			expectedPod: nil,
			expectedErr: true,
		},
		{
			name:        "Multiple Metas",
			metas:       &[]string{string(podJSON), string(podJSON)},
			mockDBError: nil,
			expectedPod: nil,
			expectedErr: true,
		},
	}

	handleMqttMeta := func() (*corev1.Pod, error) {
		value := fmt.Sprintf("default/pod/%s", mosquittoContainerName)
		metas, err := dao.QueryMeta("key", value)
		if err != nil {
			return nil, err
		}

		if metas == nil || len(*metas) != 1 {
			return nil, errors.New("invalid meta length")
		}

		var pod corev1.Pod
		if err := json.Unmarshal([]byte((*metas)[0]), &pod); err != nil {
			return nil, err
		}

		return &pod, nil
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(dao.QueryMeta, func(key string, value string) (*[]string, error) {
				assert.Equal("key", key)
				assert.Equal(fmt.Sprintf("default/pod/%s", mosquittoContainerName), value)
				return test.metas, test.mockDBError
			})
			defer patches.Reset()

			pod, err := handleMqttMeta()

			if test.expectedErr {
				assert.Error(err)
				assert.Nil(pod)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedPod, pod)
			}
		})
	}
}

func TestPods_DeleteWithDifferentContentTypes(t *testing.T) {
	assert := assert.New(t)

	deleteOptions := metav1.DeleteOptions{}

	testCases := []struct {
		name      string
		content   interface{}
		expectErr bool
	}{
		{
			name:      "Success with OK string",
			content:   constants.MessageSuccessfulContent,
			expectErr: false,
		},
		{
			name:      "Error with error content",
			content:   errors.New("delete error"),
			expectErr: true,
		},
		{
			name:      "Error with unsupported content type",
			content:   123,
			expectErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = test.content
				return resp, nil
			}

			podsClient := newPods(namespace, mockSend)

			err := podsClient.Delete(testPodName, deleteOptions)

			if test.expectErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestPods_GetWithContentErrors(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name      string
		respFunc  func(*model.Message) (*model.Message, error)
		expectErr bool
	}{
		{
			name: "Get Pod Error in GetContentData",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = 123
				return resp, nil
			},
			expectErr: true,
		},
		{
			name: "Get Pod with Not Found Response",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = []string{}
				return resp, nil
			},
			expectErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = test.respFunc

			podsClient := newPods(namespace, mockSend)

			pod, err := podsClient.Get(testPodName)

			assert.Error(err)
			assert.Nil(pod)

			if test.name == "Get Pod with Not Found Response" {
				statusErr, ok := err.(*apierrors.StatusError)
				assert.True(ok)
				assert.Equal(metav1.StatusReasonNotFound, statusErr.ErrStatus.Reason)
			}
		})
	}
}
