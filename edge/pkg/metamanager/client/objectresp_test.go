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
	"net/url"
	"path/filepath"
	"syscall"
	"testing"

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/translator"
)

// connRefusedError is the error client-go returns to cloudcore when
// kube-apiserver cannot be reached. It is not an API status error.
var connRefusedError = &url.Error{
	Op:  "Patch",
	URL: "https://127.0.0.1:6443/api/v1/namespaces/default/pods/web-0/status",
	Err: syscall.ECONNREFUSED,
}

// cloudReply builds the reply cloudcore sends for an ObjectResp and returns
// the content the edge receives after the viaduct encode/decode.
func cloudReply(t *testing.T, object metav1.Object, err error) []byte {
	t.Helper()
	msg := model.NewMessage("id").
		FillBody(&edgeapi.ObjectResp{Object: object, Err: err}).
		BuildRouter("edgecontroller", "resource", "node/edge-1/default/podpatch/web-0", model.ResponseOperation)
	raw, encErr := translator.NewTran().Encode(msg)
	if encErr != nil {
		t.Fatalf("encode: %v", encErr)
	}
	edgeMsg := &model.Message{}
	if decErr := translator.NewTran().Decode(raw, edgeMsg); decErr != nil {
		t.Fatalf("decode: %v", decErr)
	}
	content, getErr := edgeMsg.GetContentData()
	if getErr != nil {
		t.Fatalf("get content: %v", getErr)
	}
	return content
}

func storeMeta(t *testing.T, key, resType string, obj interface{}) string {
	t.Helper()
	data, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := dbclient.NewMetaService().InsertOrUpdate(&models.Meta{Key: key, Type: resType, Value: string(data)}); err != nil {
		t.Fatalf("store %s: %v", key, err)
	}
	return string(data)
}

func loadMeta(t *testing.T, key string) string {
	t.Helper()
	metas, err := dbclient.NewMetaService().QueryMeta("key", key)
	if err != nil || len(*metas) != 1 {
		t.Fatalf("query %s: err=%v, records=%v", key, err, metas)
	}
	return (*metas)[0]
}

// TestObjectRespErrors checks how the edge handles cloudcore replies for
// failed requests. dao.Init opens the DB only once per test binary, so all
// cases that need the DB run as subtests of this test.
func TestObjectRespErrors(t *testing.T) {
	dao.Init(filepath.Join(t.TempDir(), "edgecore.db"), &v1alpha2.MetaManager{Enable: true})

	const (
		podResource  = "default/podpatch/web-0"
		podKey       = "default/pod/web-0"
		nodeResource = "default/nodepatch/edge-1"
		nodeKey      = "default/node/edge-1"
	)
	pod := &corev1.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{Name: "web-0", Namespace: "default", UID: "uid-1"},
		Spec: corev1.PodSpec{
			NodeName:   "edge-1",
			Containers: []corev1.Container{{Name: "web", Image: "nginx"}},
		},
	}
	node := &corev1.Node{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Node"},
		ObjectMeta: metav1.ObjectMeta{Name: "edge-1", Labels: map[string]string{"node-role.kubernetes.io/edge": ""}},
	}
	conflict := apierrors.NewConflict(schema.GroupResource{Resource: "pods"}, "web-0", errors.New("object was modified"))

	podErrorCases := []struct {
		name string
		err  error
	}{
		// Older cloudcore sends the *url.Error as is. It decodes to an empty
		// StatusError on the edge, and the empty pod must not be stored.
		{name: "non-API error from older cloudcore", err: connRefusedError},
		// Fixed cloudcore converts it to an internal error.
		{name: "non-API error converted to InternalError", err: apierrors.NewInternalError(connRefusedError)},
		{name: "API error", err: conflict},
	}
	for _, tc := range podErrorCases {
		t.Run("pod patch: "+tc.name, func(t *testing.T) {
			before := storeMeta(t, podKey, model.ResourceTypePod, pod)

			// On failure client-go returns an empty object, and cloudcore sends it.
			got, err := handlePodResp(podResource, cloudReply(t, &corev1.Pod{}, tc.err))
			if err == nil {
				t.Fatalf("expected an error, got success with pod %q", got.Name)
			}
			if after := loadMeta(t, podKey); after != before {
				t.Errorf("local pod record was changed by a failed patch:\nbefore: %s\nafter:  %s", before, after)
			}
		})
	}

	t.Run("pod patch: success updates the local record", func(t *testing.T) {
		storeMeta(t, podKey, model.ResourceTypePod, pod)
		patched := pod.DeepCopy()
		patched.Status.Phase = corev1.PodRunning

		got, err := handlePodResp(podResource, cloudReply(t, patched, nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != pod.Name || got.Status.Phase != corev1.PodRunning {
			t.Errorf("unexpected pod returned: name=%q phase=%q", got.Name, got.Status.Phase)
		}
		var stored corev1.Pod
		if err := json.Unmarshal([]byte(loadMeta(t, podKey)), &stored); err != nil {
			t.Fatalf("unmarshal stored pod: %v", err)
		}
		if stored.Name != pod.Name || stored.Status.Phase != corev1.PodRunning {
			t.Errorf("local pod record not updated: name=%q phase=%q", stored.Name, stored.Status.Phase)
		}
	})

	nodeErrorCases := []struct {
		name string
		err  error
	}{
		{name: "non-API error from older cloudcore", err: connRefusedError},
		{name: "non-API error converted to InternalError", err: apierrors.NewInternalError(connRefusedError)},
	}
	for _, tc := range nodeErrorCases {
		t.Run("node patch: "+tc.name, func(t *testing.T) {
			before := storeMeta(t, nodeKey, model.ResourceTypeNode, node)

			got, err := handleNodeResp(nodeResource, cloudReply(t, &corev1.Node{}, tc.err))
			if err == nil {
				t.Fatalf("expected an error, got success with node %q", got.Name)
			}
			if after := loadMeta(t, nodeKey); after != before {
				t.Errorf("local node record was changed by a failed patch:\nbefore: %s\nafter:  %s", before, after)
			}
		})
	}

	t.Run("node patch: success updates the local record", func(t *testing.T) {
		storeMeta(t, nodeKey, model.ResourceTypeNode, node)
		patched := node.DeepCopy()
		patched.Status.Phase = corev1.NodeRunning

		got, err := handleNodeResp(nodeResource, cloudReply(t, patched, nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != node.Name {
			t.Errorf("unexpected node returned: %q", got.Name)
		}
		var stored corev1.Node
		if err := json.Unmarshal([]byte(loadMeta(t, nodeKey)), &stored); err != nil {
			t.Fatalf("unmarshal stored node: %v", err)
		}
		if stored.Name != node.Name || stored.Status.Phase != corev1.NodeRunning {
			t.Errorf("local node record not updated: name=%q phase=%q", stored.Name, stored.Status.Phase)
		}
	})
}

// TestHandleLeaseRespNonAPIError checks that a failed lease request is never
// returned to the caller as an empty lease.
func TestHandleLeaseRespNonAPIError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "non-API error from older cloudcore", err: connRefusedError},
		{name: "non-API error converted to InternalError", err: apierrors.NewInternalError(connRefusedError)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := handleLeaseResp(cloudReply(t, &coordinationv1.Lease{}, tc.err))
			if err == nil {
				t.Fatalf("expected an error, got success with lease %+v", got)
			}
		})
	}

	t.Run("success", func(t *testing.T) {
		lease := &coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{Name: "edge-1", Namespace: "kube-node-lease"}}
		got, err := handleLeaseResp(cloudReply(t, lease, nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != lease.Name {
			t.Errorf("lease name = %q, want %q", got.Name, lease.Name)
		}
	})
}
