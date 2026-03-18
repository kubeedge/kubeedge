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

package restart

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	fakerest "k8s.io/client-go/rest/fake"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/testutil"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

func TestPodRestart(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		resp           *types.RestartResponse
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name:   "restart success",
			status: http.StatusOK,
			resp: &types.RestartResponse{
				LogMessages: []string{"pod1 restarted"},
			},
			wantErr: false,
		},
		{
			name:           "restart failed - 500",
			status:         http.StatusInternalServerError,
			wantErr:        true,
			expectedErrMsg: "error on the server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respBytes, _ := json.Marshal(tt.resp)
			client := &fakerest.RESTClient{
				NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
				Client: fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.status,
						Body:       io.NopCloser(bytes.NewReader(respBytes)),
					}, nil
				}),
			}
			mockClient := &testutil.MockClientset{
				MockCoreV1: &testutil.MockCoreV1{
					RestClient: client,
				},
			}

			resp, err := podRestart(context.Background(), mockClient, "default", []string{"pod1"})
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.resp, resp)
			}
		})
	}
}

func TestRestartPod(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	client := &fakerest.RESTClient{
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		Client: fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
			resp := &types.RestartResponse{
				LogMessages: []string{"restarted"},
			}
			respBytes, _ := json.Marshal(resp)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(respBytes)),
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

	opts := NewRestartPodOpts()
	err := opts.restartPod([]string{"pod1"})
	assert.NoError(t, err)
}

func TestNewEdgePodRestart(t *testing.T) {
	assert := assert.New(t)
	cmd := NewEdgePodRestart()

	assert.NotNil(cmd)
	assert.Equal("pod", cmd.Use)
	assert.Equal(edgePodRestartShortDescription, cmd.Short)
	assert.Equal(edgePodRestartShortDescription, cmd.Long)

	assert.NotNil(cmd.RunE)

	assert.Equal(cmd.Flags().Lookup(common.FlagNameNamespace).Name, "namespace")
}

func TestNewRestartPodOpts(t *testing.T) {
	assert := assert.New(t)

	podRestartOptions := NewRestartPodOpts()
	assert.NotNil(podRestartOptions)
	assert.Equal(podRestartOptions.Namespace, "default")
}

func TestAddRestartPodFlags(t *testing.T) {
	assert := assert.New(t)
	getOptions := NewRestartPodOpts()

	cmd := &cobra.Command{}

	AddRestartPodFlags(cmd, getOptions)

	namespaceFlag := cmd.Flags().Lookup(common.FlagNameNamespace)
	assert.Equal("default", namespaceFlag.DefValue)
	assert.Equal("namespace", namespaceFlag.Name)
}
