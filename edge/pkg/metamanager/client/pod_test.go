/*
Copyright 2026 The KubeEdge Authors.

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
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

const (
	testPodNamespace = "default"
	testPodName      = "test-pod"
	testPodKey       = "default/pod/test-pod"
)

func TestPodsCreate(t *testing.T) {
	t.Run("pod is created", func(t *testing.T) {
		saved := patchPodMetaDB(t, nil)
		c, sent := newTestPods(replyWith(model.ResponseOperation, PodResp{Object: newTestPod()}))

		pod, err := c.Create(newTestPod())
		require.NoError(t, err)
		assert.Equal(t, testPodName, pod.Name)
		assertPodRequest(t, *sent, testPodKey, model.InsertOperation)
		// The created pod is stored in the edge meta db.
		assertPodMeta(t, *saved, testPodKey)
	})

	t.Run("create is rejected", func(t *testing.T) {
		saved := patchPodMetaDB(t, nil)
		alreadyExists := apierrors.NewAlreadyExists(schema.GroupResource{Resource: "pods"}, testPodName)
		c, _ := newTestPods(replyWith(model.ResponseOperation, PodResp{Err: *alreadyExists}))

		_, err := c.Create(newTestPod())
		assert.True(t, apierrors.IsAlreadyExists(err), "got %v", err)
		assert.Empty(t, *saved)
	})

	t.Run("failed to store the pod", func(t *testing.T) {
		patchPodMetaDB(t, errors.New("test error"))
		c, _ := newTestPods(replyWith(model.ResponseOperation, PodResp{Object: newTestPod()}))

		_, err := c.Create(newTestPod())
		assert.ErrorContains(t, err, "update pod meta failed, err: test error")
	})

	t.Run("failed to send", func(t *testing.T) {
		c, _ := newTestPods(replyError(errors.New("test error")))
		_, err := c.Create(newTestPod())
		assert.ErrorContains(t, err, "create pod failed, err: test error")
	})

	t.Run("invalid response", func(t *testing.T) {
		c, _ := newTestPods(replyWith(model.ResponseOperation, make(chan int)))
		_, err := c.Create(newTestPod())
		assert.ErrorContains(t, err, "parse message to pod failed")

		c, _ = newTestPods(replyWith(model.ResponseOperation, "not a pod response"))
		_, err = c.Create(newTestPod())
		assert.ErrorContains(t, err, "unmarshal message to pod failed")
	})
}

func TestPodsUpdate(t *testing.T) {
	c, sent := newTestPods(replyError(errors.New("must not be called")))
	assert.NoError(t, c.Update(newTestPod()))
	assert.Empty(t, *sent)
}

func TestPodsDelete(t *testing.T) {
	options := metav1.DeleteOptions{GracePeriodSeconds: new(int64)}

	t.Run("pod is deleted", func(t *testing.T) {
		c, sent := newTestPods(replyWith(model.ResponseOperation, constants.MessageSuccessfulContent))
		require.NoError(t, c.Delete(testPodName, options))
		assertPodRequest(t, *sent, testPodKey, model.DeleteOperation)
		assert.Equal(t, options, (*sent)[0].GetContent())
	})

	t.Run("delete returns an error", func(t *testing.T) {
		c, _ := newTestPods(replyWith(model.ResponseOperation, errors.New("test error")))
		assert.EqualError(t, c.Delete(testPodName, options), "test error")
	})

	t.Run("unsupported response", func(t *testing.T) {
		// Only the successful content string means the pod is deleted.
		for _, content := range []interface{}{"unexpected", 1} {
			c, _ := newTestPods(replyWith(model.ResponseOperation, content))
			assert.EqualError(t, c.Delete(testPodName, options), "delete pod failed, response content type unsupported")
		}
	})

	t.Run("failed to send", func(t *testing.T) {
		c, _ := newTestPods(replyError(errors.New("test error")))
		assert.EqualError(t, c.Delete(testPodName, options), "test error")
	})
}

func TestPodsGet(t *testing.T) {
	podJSON, err := json.Marshal(newTestPod())
	require.NoError(t, err)

	t.Run("pod is found", func(t *testing.T) {
		c, sent := newTestPods(replyWith(model.ResponseOperation, []string{string(podJSON)}))
		pod, err := c.Get(testPodName)
		require.NoError(t, err)
		assert.Equal(t, newTestPod(), pod)
		assertPodRequest(t, *sent, testPodKey, model.QueryOperation)
	})

	t.Run("pod is not found", func(t *testing.T) {
		c, _ := newTestPods(replyWith(model.ResponseOperation, []string{}))
		_, err := c.Get(testPodName)
		assert.True(t, apierrors.IsNotFound(err), "got %v", err)
		assert.ErrorContains(t, err, testPodName)
	})

	t.Run("more than one pod", func(t *testing.T) {
		c, _ := newTestPods(replyWith(model.ResponseOperation, []string{string(podJSON), string(podJSON)}))
		_, err := c.Get(testPodName)
		assert.EqualError(t, err, "pod length from meta db is 2")
	})

	t.Run("invalid response", func(t *testing.T) {
		c, _ := newTestPods(replyWith(model.ResponseOperation, make(chan int)))
		_, err := c.Get(testPodName)
		assert.ErrorContains(t, err, "parse message to pod failed")

		c, _ = newTestPods(replyWith(model.ResponseOperation, "not a list"))
		_, err = c.Get(testPodName)
		assert.ErrorContains(t, err, "unmarshal message to pod list from db failed")

		c, _ = newTestPods(replyWith(model.ResponseOperation, []string{"not a pod"}))
		_, err = c.Get(testPodName)
		assert.ErrorContains(t, err, "unmarshal message to pod failed")
	})

	t.Run("failed to send", func(t *testing.T) {
		c, _ := newTestPods(replyError(errors.New("test error")))
		_, err := c.Get(testPodName)
		assert.ErrorContains(t, err, "get pod failed, err: test error")
	})
}

func TestPodsPatch(t *testing.T) {
	patch := []byte(`{"metadata":{"labels":{"app":"test"}}}`)

	t.Run("pod is patched", func(t *testing.T) {
		saved := patchPodMetaDB(t, nil)
		c, sent := newTestPods(replyWith(model.ResponseOperation, PodResp{Object: newTestPod()}))

		pod, err := c.Patch(testPodName, patch)
		require.NoError(t, err)
		assert.Equal(t, testPodName, pod.Name)
		assertPodRequest(t, *sent, "default/podpatch/test-pod", model.PatchOperation)
		assert.Equal(t, string(patch), (*sent)[0].GetContent())
		// The patched pod is stored under its pod key, not the podpatch resource.
		assertPodMeta(t, *saved, testPodKey)
	})

	t.Run("patch is rejected", func(t *testing.T) {
		saved := patchPodMetaDB(t, nil)
		c, _ := newTestPods(replyWith(model.ResponseErrorOperation, "patch rejected"))

		_, err := c.Patch(testPodName, patch)
		assert.EqualError(t, err, "patch rejected")
		assert.Empty(t, *saved)
	})

	t.Run("failed to send", func(t *testing.T) {
		c, _ := newTestPods(replyError(errors.New("test error")))
		_, err := c.Patch(testPodName, patch)
		assert.ErrorContains(t, err, "update pod failed, err: test error")
	})

	t.Run("invalid response", func(t *testing.T) {
		c, _ := newTestPods(replyWith(model.ResponseOperation, make(chan int)))
		_, err := c.Patch(testPodName, patch)
		assert.ErrorContains(t, err, "parse message to pod failed")
	})
}

func newTestPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: testPodNamespace, Name: testPodName},
		Spec:       corev1.PodSpec{NodeName: "edge-node"},
	}
}

// newTestPods returns a pods client whose requests are answered by reply,
// and the requests it sends.
func newTestPods(reply func(*model.Message) (*model.Message, error)) (*pods, *[]*model.Message) {
	var sent []*model.Message
	send := &mockSendInterface{
		sendSyncFunc: func(msg *model.Message) (*model.Message, error) {
			sent = append(sent, msg)
			return reply(msg)
		},
	}
	return newPods(testPodNamespace, send), &sent
}

func replyWith(operation string, content interface{}) func(*model.Message) (*model.Message, error) {
	return func(req *model.Message) (*model.Message, error) {
		return model.NewMessage(req.GetID()).
			BuildRouter(modules.MetaManagerModuleName, modules.MetaGroup, req.GetResource(), operation).
			FillBody(content), nil
	}
}

func replyError(err error) func(*model.Message) (*model.Message, error) {
	return func(*model.Message) (*model.Message, error) {
		return nil, err
	}
}

// patchPodMetaDB records the pods stored in the edge meta db and fails with err.
func patchPodMetaDB(t *testing.T, err error) *[]models.Meta {
	var saved []models.Meta
	patches := gomonkey.ApplyMethodFunc(reflect.TypeOf((*dbclient.MetaService)(nil)), "InsertOrUpdate",
		func(meta *models.Meta) error {
			saved = append(saved, *meta)
			return err
		})
	t.Cleanup(patches.Reset)
	return &saved
}

func assertPodRequest(t *testing.T, sent []*model.Message, resource, operation string) {
	t.Helper()
	require.Len(t, sent, 1)
	assert.Equal(t, modules.EdgedModuleName, sent[0].GetSource())
	assert.Equal(t, modules.MetaGroup, sent[0].GetGroup())
	assert.Equal(t, resource, sent[0].GetResource())
	assert.Equal(t, operation, sent[0].GetOperation())
}

func assertPodMeta(t *testing.T, saved []models.Meta, key string) {
	t.Helper()
	require.Len(t, saved, 1)
	assert.Equal(t, key, saved[0].Key)
	assert.Equal(t, model.ResourceTypePod, saved[0].Type)
	var pod corev1.Pod
	require.NoError(t, json.Unmarshal([]byte(saved[0].Value), &pod))
	assert.Equal(t, "v1", pod.APIVersion)
	assert.Equal(t, "Pod", pod.Kind)
	assert.Equal(t, testPodName, pod.Name)
}
