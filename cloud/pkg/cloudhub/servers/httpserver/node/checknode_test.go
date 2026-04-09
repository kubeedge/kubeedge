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

package node

// Note: Tests in this file use gomonkey for function patching.
// Run tests with inlining disabled:
// go test -gcflags=all=-l ./cloud/pkg/cloudhub/servers/httpserver/node/...

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
)

func TestCheckNode(t *testing.T) {
	tests := []struct {
		name           string
		nodeName       string
		existingNodes  []string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "node not found returns 404",
			nodeName:       "nonexistent-node",
			existingNodes:  []string{},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Node not found",
		},
		{
			name:           "node exists returns 200",
			nodeName:       "existing-node",
			existingNodes:  []string{"existing-node"},
			expectedStatus: http.StatusOK,
			expectedBody:   "Node found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset()
			for _, name := range tt.existingNodes {
				node := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
					},
				}
				_, err := fakeClient.CoreV1().Nodes().
					Create(context.TODO(), node, metav1.CreateOptions{})
				assert.NoError(t, err)
			}

			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(client.GetKubeClient, kubernetes.Interface(fakeClient))

			ws := new(restful.WebService)
			ws.Route(ws.GET("/checknode/{nodename}").To(CheckNode))
			container := restful.NewContainer()
			container.Add(ws)

			httpReq := httptest.NewRequest(http.MethodGet,
				"/checknode/"+tt.nodeName, nil)
			httpWriter := httptest.NewRecorder()
			container.ServeHTTP(httpWriter, httpReq)

			assert.Equal(t, tt.expectedStatus, httpWriter.Code)
			assert.Contains(t, httpWriter.Body.String(), tt.expectedBody)
		})
	}
}

func TestCheckNode_EmptyNodeName(t *testing.T) {
	httpReq := httptest.NewRequest(http.MethodGet, "/", nil)
	httpWriter := httptest.NewRecorder()

	req := restful.NewRequest(httpReq)
	resp := restful.NewResponse(httpWriter)

	CheckNode(req, resp)

	assert.Equal(t, http.StatusBadRequest, httpWriter.Code)
	assert.Contains(t, httpWriter.Body.String(), "nodename parameter is required")
}

func TestCheckNode_InternalServerError(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fakeClient.Fake.PrependReactor("get", "nodes",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf("internal server error")
		})

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFuncReturn(client.GetKubeClient, kubernetes.Interface(fakeClient))

	ws := new(restful.WebService)
	ws.Route(ws.GET("/checknode/{nodename}").To(CheckNode))
	container := restful.NewContainer()
	container.Add(ws)

	httpReq := httptest.NewRequest(http.MethodGet, "/checknode/test-node", nil)
	httpWriter := httptest.NewRecorder()
	container.ServeHTTP(httpWriter, httpReq)

	assert.Equal(t, http.StatusInternalServerError, httpWriter.Code)
	assert.Contains(t, httpWriter.Body.String(), "Failed to query node information")
}
