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

package unhold

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/testutil"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

func TestNewEdgeUnholdUpgrade(t *testing.T) {
	assert := assert.New(t)
	cmd := NewEdgeUnholdUpgrade()

	assert.NotNil(cmd)
	assert.Equal("unhold-upgrade <resource-type> [<name>] [--namespace namespace]", cmd.Use)
	assert.Equal("Unhold an upgrade for a pod or node-wide", cmd.Short)
	assert.NotNil(cmd.RunE)
	assert.Equal("namespace", cmd.Flags().Lookup("namespace").Name)
}

func TestNewEdgeUnholdUpgradePodRequiresName(t *testing.T) {
	cmd := newSilencedUnholdCommand()
	cmd.SetArgs([]string{"pod"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pod name is required")
}

func TestNewEdgeUnholdUpgradeUnknownResourceType(t *testing.T) {
	cmd := newSilencedUnholdCommand()
	cmd.SetArgs([]string{"service"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type: service")
}

func TestNewEdgeUnholdUpgradePodSuccess(t *testing.T) {
	var method, path, contentType, requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		contentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requestBody = string(body)
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

	cmd := newSilencedUnholdCommand()
	_ = cmd.Flags().Set("namespace", "test-ns")
	cmd.SetArgs([]string{"pod", "test-pod"})

	err = cmd.Execute()

	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, method)
	assert.Equal(t, "/api/v1/pods/unhold-upgrade", path)
	assert.Equal(t, "text/plain", contentType)
	assert.Equal(t, "test-ns/test-pod", requestBody)
}

func TestNewEdgeUnholdUpgradeNodeUsesCurrentNodeName(t *testing.T) {
	var path, requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requestBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	clientset, err := testutil.NewCoreV1Clientset(server.URL, server.Client())
	require.NoError(t, err)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(getCurrentNodeName, func() (string, error) {
		return "edge-node-1", nil
	})
	patches.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
		return clientset, nil
	})

	cmd := newSilencedUnholdCommand()
	cmd.SetArgs([]string{"node"})

	err = cmd.Execute()

	assert.NoError(t, err)
	assert.Equal(t, "/api/v1/nodes/unhold-upgrade", path)
	assert.Equal(t, "edge-node-1", requestBody)
}

func TestNewEdgeUnholdUpgradeNodeLookupError(t *testing.T) {
	patches := gomonkey.ApplyFunc(getCurrentNodeName, func() (string, error) {
		return "", errors.New("node lookup failed")
	})
	defer patches.Reset()

	cmd := newSilencedUnholdCommand()
	cmd.SetArgs([]string{"node"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "node lookup failed")
}

func TestUnholdResourceUpgradeKubeClientError(t *testing.T) {
	patches := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
		return nil, errors.New("client init failed")
	})
	defer patches.Reset()

	err := unholdResourceUpgrade("pods", "default/test-pod")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "client init failed")
}

func TestUnholdResourceUpgradeStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		require.NoError(t, json.NewEncoder(w).Encode(&metav1.Status{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Status",
				APIVersion: "v1",
			},
			Status:  metav1.StatusFailure,
			Message: "upgrade is not on hold",
			Code:    http.StatusConflict,
		}))
	}))
	defer server.Close()

	clientset, err := testutil.NewCoreV1Clientset(server.URL, server.Client())
	require.NoError(t, err)

	patches := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
		return clientset, nil
	})
	defer patches.Reset()

	err = unholdResourceUpgrade("pods", "default/test-pod")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unhold pods upgrade")
	assert.Contains(t, err.Error(), "status code: 409")
	assert.Contains(t, err.Error(), "message: upgrade is not on hold")
}

func TestUnholdResourceUpgradeGenericError(t *testing.T) {
	clientset, err := testutil.NewCoreV1Clientset("http://example.invalid", &http.Client{
		Transport: testutil.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("request failed")
		}),
	})
	require.NoError(t, err)

	patches := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
		return clientset, nil
	})
	defer patches.Reset()

	err = unholdResourceUpgrade("nodes", "edge-node-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send unhold request to MetaService API")
	assert.Contains(t, err.Error(), "request failed")
}

func newSilencedUnholdCommand() *cobra.Command {
	cmd := NewEdgeUnholdUpgrade()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}
