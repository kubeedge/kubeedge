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

package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
)

// edgeObjectResp has the shape the edge uses to decode an ObjectResp
// (see PodResp, NodeResp and LeaseResp in edge/pkg/metamanager/client).
type edgeObjectResp struct {
	Object json.RawMessage
	Err    apierrors.StatusError
}

// newConnRefusedError returns the error client-go returns when
// kube-apiserver cannot be reached.
func newConnRefusedError() error {
	return &url.Error{
		Op:  "Patch",
		URL: "https://127.0.0.1:6443/api/v1/namespaces/default/pods/web-0/status",
		Err: syscall.ECONNREFUSED,
	}
}

// isEmptyStatusError is the check the edge uses to decide that a request succeeded.
func isEmptyStatusError(err apierrors.StatusError) bool {
	return reflect.DeepEqual(err, apierrors.StatusError{})
}

func decodeOnEdge(t *testing.T, resp *edgeapi.ObjectResp) edgeObjectResp {
	t.Helper()
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal ObjectResp: %v", err)
	}
	var out edgeObjectResp
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal ObjectResp on the edge: %v", err)
	}
	return out
}

func TestToAPIStatusError(t *testing.T) {
	conflict := apierrors.NewConflict(schema.GroupResource{Resource: "pods"}, "web-0", errors.New("object was modified"))
	connRefused := newConnRefusedError()

	tests := []struct {
		name        string
		err         error
		wantNil     bool
		wantSame    error // the exact error that must be returned, if set
		wantCode    int32
		wantReason  metav1.StatusReason
		wantMessage string
	}{
		{
			name:    "nil stays nil",
			err:     nil,
			wantNil: true,
		},
		{
			name:       "StatusError is returned unchanged",
			err:        conflict,
			wantSame:   conflict,
			wantCode:   http.StatusConflict,
			wantReason: metav1.StatusReasonConflict,
		},
		{
			name:       "wrapped StatusError is unwrapped",
			err:        fmt.Errorf("patch pod: %w", conflict),
			wantSame:   conflict,
			wantCode:   http.StatusConflict,
			wantReason: metav1.StatusReasonConflict,
		},
		{
			name:        "url.Error becomes an internal error",
			err:         connRefused,
			wantCode:    http.StatusInternalServerError,
			wantReason:  metav1.StatusReasonInternalError,
			wantMessage: connRefused.Error(),
		},
		{
			name:        "plain error becomes an internal error",
			err:         errors.New("failed to get cloudcore localIP"),
			wantCode:    http.StatusInternalServerError,
			wantReason:  metav1.StatusReasonInternalError,
			wantMessage: "failed to get cloudcore localIP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toAPIStatusError(tt.err)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %#v", got)
				}
				return
			}
			statusErr, ok := got.(*apierrors.StatusError)
			if !ok {
				t.Fatalf("expected *StatusError, got %T", got)
			}
			if tt.wantSame != nil && got != tt.wantSame {
				t.Errorf("expected the original StatusError to be returned, got %#v", got)
			}
			if statusErr.ErrStatus.Code != tt.wantCode {
				t.Errorf("code = %d, want %d", statusErr.ErrStatus.Code, tt.wantCode)
			}
			if statusErr.ErrStatus.Reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", statusErr.ErrStatus.Reason, tt.wantReason)
			}
			if !strings.Contains(statusErr.ErrStatus.Message, tt.wantMessage) {
				t.Errorf("message %q does not contain %q", statusErr.ErrStatus.Message, tt.wantMessage)
			}
		})
	}
}

// TestObjectRespNonAPIErrorReachesEdge checks the JSON the edge receives: a
// non-API error must decode into a non-empty StatusError on the edge, because
// the edge treats an empty StatusError as success.
func TestObjectRespNonAPIErrorReachesEdge(t *testing.T) {
	connRefused := newConnRefusedError()

	// Without the conversion the edge cannot see the error at all.
	raw := decodeOnEdge(t, &edgeapi.ObjectResp{Object: &corev1.Pod{}, Err: connRefused})
	if !isEmptyStatusError(raw.Err) {
		t.Fatalf("expected a raw *url.Error to decode to an empty StatusError, got %#v", raw.Err)
	}

	got := decodeOnEdge(t, &edgeapi.ObjectResp{Object: &corev1.Pod{}, Err: toAPIStatusError(connRefused)})
	if isEmptyStatusError(got.Err) {
		t.Fatalf("edge decoded an empty StatusError, so the error would be treated as success")
	}
	if !apierrors.IsInternalError(&got.Err) {
		t.Errorf("expected an internal error on the edge, got %#v", got.Err.ErrStatus)
	}
	if !strings.Contains(got.Err.Error(), connRefused.Error()) {
		t.Errorf("edge error %q does not contain the original error %q", got.Err.Error(), connRefused.Error())
	}
}

// responseLayer is a race-free MessageLayer that hands responses to the test.
type responseLayer struct {
	responses chan model.Message
}

func (l *responseLayer) Send(model.Message) error { return nil }

func (l *responseLayer) Receive() (model.Message, error) {
	return model.Message{}, errors.New("not implemented")
}

func (l *responseLayer) Response(message model.Message) error {
	l.responses <- message
	return nil
}

// TestUpstreamNonAPIErrorResponses runs the real handlers with a client that
// fails like client-go does when kube-apiserver cannot be reached: it returns
// an empty object and a *url.Error.
func TestUpstreamNonAPIErrorResponses(t *testing.T) {
	tests := []struct {
		name     string
		verb     string
		resource string
		empty    runtime.Object
		start    func(uc *UpstreamController) chan model.Message
		msg      model.Message
	}{
		{
			name:     "patch pod",
			verb:     "patch",
			resource: "pods",
			empty:    &corev1.Pod{},
			start: func(uc *UpstreamController) chan model.Message {
				go uc.patchPod()
				return uc.patchPodChan
			},
			msg: model.Message{
				Header: model.MessageHeader{ID: "patch-pod"},
				Router: model.MessageRoute{
					Resource:  "node/edge-1/default/" + model.ResourceTypePodPatch + "/web-0",
					Operation: model.PatchOperation,
				},
				Content: `{"status":{"phase":"Running"}}`,
			},
		},
		{
			name:     "patch node",
			verb:     "patch",
			resource: "nodes",
			empty:    &corev1.Node{},
			start: func(uc *UpstreamController) chan model.Message {
				go uc.patchNode()
				return uc.patchNodeChan
			},
			msg: model.Message{
				Header: model.MessageHeader{ID: "patch-node"},
				Router: model.MessageRoute{
					Resource:  "node/edge-1/default/" + model.ResourceTypeNodePatch + "/edge-1",
					Operation: model.PatchOperation,
				},
				Content: `{"status":{}}`,
			},
		},
		{
			name:     "update lease",
			verb:     "update",
			resource: "leases",
			empty:    &coordinationv1.Lease{},
			start: func(uc *UpstreamController) chan model.Message {
				go uc.createOrUpdateLease()
				return uc.createLeaseChan
			},
			msg: model.Message{
				Header: model.MessageHeader{ID: "update-lease"},
				Router: model.MessageRoute{
					Resource:  "node/edge-1/kube-node-lease/" + model.ResourceTypeLease + "/edge-1",
					Operation: model.UpdateOperation,
				},
				Content: `{"metadata":{"name":"edge-1","namespace":"kube-node-lease"}}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient := fake.NewSimpleClientset()
			kubeClient.PrependReactor(tt.verb, tt.resource, func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, tt.empty, newConnRefusedError()
			})
			layer := &responseLayer{responses: make(chan model.Message, 1)}
			uc := &UpstreamController{
				kubeClient:      kubeClient,
				messageLayer:    layer,
				patchPodChan:    make(chan model.Message, 1),
				patchNodeChan:   make(chan model.Message, 1),
				createLeaseChan: make(chan model.Message, 1),
			}

			tt.start(uc) <- tt.msg

			var resp model.Message
			select {
			case resp = <-layer.responses:
			case <-time.After(5 * time.Second):
				t.Fatalf("no response sent to the edge")
			}
			if resp.GetParentID() != tt.msg.GetID() {
				t.Fatalf("response parent ID = %q, want %q", resp.GetParentID(), tt.msg.GetID())
			}
			objResp, ok := resp.GetContent().(*edgeapi.ObjectResp)
			if !ok {
				t.Fatalf("response content is %T, want *ObjectResp", resp.GetContent())
			}
			got := decodeOnEdge(t, objResp)
			if isEmptyStatusError(got.Err) {
				t.Fatalf("edge decoded an empty StatusError, so the failed %s would be treated as success", tt.name)
			}
			if !apierrors.IsInternalError(&got.Err) {
				t.Errorf("expected an internal error on the edge, got %#v", got.Err.ErrStatus)
			}
		})
	}
}
