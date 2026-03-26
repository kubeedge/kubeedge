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

package confirm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/testutil"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

func TestNewEdgeConfirm(t *testing.T) {
	assert := assert.New(t)
	cmd := NewEdgeConfirm()

	assert.NotNil(cmd)
	assert.Equal("confirm", cmd.Use)
	assert.Equal("Send a confirmation signal to the MetaService API.", cmd.Short)
	assert.NotNil(cmd.RunE)
}

func TestNewEdgeConfirmRunEKubeClientError(t *testing.T) {
	patches := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
		return nil, errors.New("client init failed")
	})
	defer patches.Reset()

	cmd := NewEdgeConfirm()
	err := cmd.RunE(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "client init failed")
}

func TestNewEdgeConfirmRunESuccess(t *testing.T) {
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	clientset, err := testutil.NewCoreV1Clientset(server.URL, server.Client())
	require.NoError(t, err)

	patches := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
		return clientset, nil
	})
	defer patches.Reset()

	cmd := NewEdgeConfirm()
	err = cmd.RunE(cmd, nil)

	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, method)
	assert.Equal(t, "/api/v1/taskupgrade/confirm-upgrade", path)
}

func TestConfirmNodeUpgradeStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		require.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Status",
				APIVersion: "v1",
			},
			Status:  metav1.StatusFailure,
			Message: "upgrade task not found",
			Code:    http.StatusBadRequest,
		}))
	}))
	defer server.Close()

	clientset, err := testutil.NewCoreV1Clientset(server.URL, server.Client())
	require.NoError(t, err)

	err = confirmNodeUpgrade(context.Background(), clientset)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to confirm node upgrade")
	assert.Contains(t, err.Error(), "status code: 400")
	assert.Contains(t, err.Error(), "message: upgrade task not found")
}

func TestConfirmNodeUpgradeGenericError(t *testing.T) {
	clientset, err := testutil.NewCoreV1Clientset("http://example.invalid", &http.Client{
		Transport: testutil.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("request failed")
		}),
	})
	require.NoError(t, err)

	err = confirmNodeUpgrade(context.Background(), clientset)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send confirm request to MetaService API")
	assert.Contains(t, err.Error(), "request failed")
}
