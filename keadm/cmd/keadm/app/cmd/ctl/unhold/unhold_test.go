/*
Copyright 2025 The KubeEdge Authors.

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

package unhold

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	fakerest "k8s.io/client-go/rest/fake"
	"k8s.io/kubectl/pkg/scheme"

	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/testutil"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

func TestNewEdgeUnholdUpgrade(t *testing.T) {
	cmd := NewEdgeUnholdUpgrade()
	assert.NotNil(t, cmd)
	assert.Equal(t, "unhold-upgrade <resource-type> [<name>] [--namespace namespace]", cmd.Use)
}

func TestUnholdResourceUpgrade(t *testing.T) {
	tests := []struct {
		name           string
		resource       string
		target         string
		status         int
		expectedErrMsg string
		wantErr        bool
		kubeClientErr  bool
	}{
		{
			name:     "unhold success",
			resource: "pods",
			target:   "default/test-pod",
			status:   http.StatusOK,
			wantErr:  false,
		},
		{
			name:           "unhold failed - kube client error",
			resource:       "pods",
			target:         "default/test-pod",
			wantErr:        true,
			kubeClientErr:  true,
			expectedErrMsg: "kube client error",
		},
		{
			name:           "unhold failed - status error",
			resource:       "nodes",
			target:         "test-node",
			status:         http.StatusInternalServerError,
			wantErr:        true,
			expectedErrMsg: "failed to unhold nodes upgrade, status code: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			if tt.kubeClientErr {
				patches.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
					return nil, fmt.Errorf("kube client error")
				})
			} else {
				client := &fakerest.RESTClient{
					NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
					Client: fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: tt.status,
							Body:       http.NoBody,
						}, nil
					}),
				}
				mockClient := &testutil.MockClientset{
					MockCoreV1: &testutil.MockCoreV1{
						RestClient: client,
					},
				}
				patches.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
					return mockClient, nil
				})
			}

			err := unholdResourceUpgrade(tt.resource, tt.target)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetCurrentNodeName(t *testing.T) {
	tests := []struct {
		name     string
		config   *cfgv1alpha2.EdgeCoreConfig
		wantErr  bool
		err      error
		expected string
	}{
		{
			name: "get node name success",
			config: &cfgv1alpha2.EdgeCoreConfig{
				Modules: &cfgv1alpha2.Modules{
					Edged: &cfgv1alpha2.Edged{
						TailoredKubeletFlag: cfgv1alpha2.TailoredKubeletFlag{
							HostnameOverride: "test-node",
						},
					},
				},
			},
			expected: "test-node",
			wantErr:  false,
		},
		{
			name:    "parse config error",
			err:     fmt.Errorf("parse error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(util.ParseEdgecoreConfig, func(path string) (*cfgv1alpha2.EdgeCoreConfig, error) {
				return tt.config, tt.err
			})

			nodeName, err := getCurrentNodeName()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, nodeName)
			}
		})
	}
}
