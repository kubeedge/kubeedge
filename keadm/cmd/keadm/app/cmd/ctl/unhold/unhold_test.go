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
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	restfake "k8s.io/client-go/rest/fake"

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
	patches := gomonkey.ApplyFunc(unholdResourceUpgrade, func(resource, target string) error {
		assert.Equal(t, "pods", resource)
		assert.Equal(t, "test-ns/test-pod", target)
		return nil
	})
	defer patches.Reset()

	cmd := NewEdgeUnholdUpgrade()
	_ = cmd.Flags().Set("namespace", "test-ns")

	err := cmd.RunE(cmd, []string{"pod", "test-pod"})

	assert.NoError(t, err)
}

func TestNewEdgeUnholdUpgradeNodeUsesCurrentNodeName(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(getCurrentNodeName, func() (string, error) {
		return "edge-node-1", nil
	})
	patches.ApplyFunc(unholdResourceUpgrade, func(resource, target string) error {
		assert.Equal(t, "nodes", resource)
		assert.Equal(t, "edge-node-1", target)
		return nil
	})

	cmd := NewEdgeUnholdUpgrade()

	err := cmd.RunE(cmd, []string{"node"})

	assert.NoError(t, err)
}

func TestNewEdgeUnholdUpgradeNodeLookupError(t *testing.T) {
	patches := gomonkey.ApplyFunc(getCurrentNodeName, func() (string, error) {
		return "", errors.New("node lookup failed")
	})
	defer patches.Reset()

	cmd := NewEdgeUnholdUpgrade()

	err := cmd.RunE(cmd, []string{"node"})

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

func TestUnholdResourceUpgradeWithRESTClientSuccess(t *testing.T) {
	var (
		method      string
		path        string
		contentType string
		requestBody string
	)
	restClient := newFakeRESTClient(func(req *http.Request) (*http.Response, error) {
		method = req.Method
		path = req.URL.Path
		contentType = req.Header.Get("Content-Type")
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		requestBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
			Header:     make(http.Header),
		}, nil
	})

	err := unholdResourceUpgradeWithRESTClient(context.Background(), restClient, "pods", "default/test-pod")

	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, method)
	assert.Equal(t, "/api/pods/unhold-upgrade", path)
	assert.Equal(t, "text/plain", contentType)
	assert.Equal(t, "default/test-pod", requestBody)
}

func TestUnholdResourceUpgradeWithRESTClientStatusError(t *testing.T) {
	restClient := &restfake.RESTClient{
		Err: &apierrors.StatusError{
			ErrStatus: metav1.Status{
				Status:  metav1.StatusFailure,
				Message: "upgrade is not on hold",
				Code:    http.StatusConflict,
			},
		},
		GroupVersion:         schema.GroupVersion{Version: "v1"},
		NegotiatedSerializer: k8sscheme.Codecs.WithoutConversion(),
		VersionedAPIPath:     "/api",
	}

	err := unholdResourceUpgradeWithRESTClient(context.Background(), restClient, "pods", "default/test-pod")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unhold pods upgrade")
	assert.Contains(t, err.Error(), "status code: 409")
	assert.Contains(t, err.Error(), "message: upgrade is not on hold")
}

func TestUnholdResourceUpgradeWithRESTClientGenericError(t *testing.T) {
	restClient := &restfake.RESTClient{
		Err:                  errors.New("request failed"),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
		NegotiatedSerializer: k8sscheme.Codecs.WithoutConversion(),
		VersionedAPIPath:     "/api",
	}

	err := unholdResourceUpgradeWithRESTClient(context.Background(), restClient, "nodes", "edge-node-1")

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

func newFakeRESTClient(roundTripper func(*http.Request) (*http.Response, error)) *restfake.RESTClient {
	return &restfake.RESTClient{
		Client:               restfake.CreateHTTPClient(roundTripper),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
		NegotiatedSerializer: k8sscheme.Codecs.WithoutConversion(),
		VersionedAPIPath:     "/api",
	}
}
