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

package confirm

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	fakerest "k8s.io/client-go/rest/fake"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/testutil"
)

func TestNewEdgeConfirm(t *testing.T) {
	cmd := NewEdgeConfirm()
	assert.NotNil(t, cmd)
	assert.Equal(t, "confirm", cmd.Use)
}

func TestConfirmNodeUpgrade(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		expectedErrMsg string
		wantErr        bool
		connErr        bool
	}{
		{
			name:    "confirm success",
			status:  http.StatusOK,
			wantErr: false,
		},
		{
			name:           "confirm failed with 500 status error",
			status:         http.StatusInternalServerError,
			wantErr:        true,
			expectedErrMsg: "failed to confirm node upgrade, status code: 500",
		},
		{
			name:           "confirm failed with connection error",
			wantErr:        true,
			expectedErrMsg: "failed to send confirm request to MetaService API",
			connErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakerest.RESTClient{
				NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
				Client: fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
					if tt.connErr {
						return nil, fmt.Errorf("connection refused")
					}
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

			err := confirmNodeUpgrade(context.Background(), mockClient)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
