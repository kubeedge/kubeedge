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
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
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

func TestNewEdgeConfirmRunEConfirmError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
		return nil, nil
	})
	patches.ApplyFunc(confirmNodeUpgrade, func(_ context.Context, _ kubernetes.Interface) error {
		return errors.New("confirm failed")
	})

	cmd := NewEdgeConfirm()
	err := cmd.RunE(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "confirm failed")
}

func TestConfirmNodeUpgradeWithRESTClientSuccess(t *testing.T) {
	var (
		method string
		path   string
	)
	restClient := newFakeRESTClient(func(req *http.Request) (*http.Response, error) {
		method = req.Method
		path = req.URL.Path
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
			Header:     make(http.Header),
		}, nil
	})

	err := confirmNodeUpgradeWithRESTClient(context.Background(), restClient)

	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, method)
	assert.Equal(t, "/api/taskupgrade/confirm-upgrade", path)
}

func TestConfirmNodeUpgradeWithRESTClientStatusError(t *testing.T) {
	restClient := &restfake.RESTClient{
		Err: &apierrors.StatusError{
			ErrStatus: metav1.Status{
				Status:  metav1.StatusFailure,
				Message: "upgrade task not found",
				Code:    http.StatusBadRequest,
			},
		},
		GroupVersion:         schema.GroupVersion{Version: "v1"},
		NegotiatedSerializer: k8sscheme.Codecs.WithoutConversion(),
		VersionedAPIPath:     "/api",
	}

	err := confirmNodeUpgradeWithRESTClient(context.Background(), restClient)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to confirm node upgrade")
	assert.Contains(t, err.Error(), "status code: 400")
	assert.Contains(t, err.Error(), "message: upgrade task not found")
}

func TestConfirmNodeUpgradeWithRESTClientGenericError(t *testing.T) {
	restClient := &restfake.RESTClient{
		Err:                  errors.New("request failed"),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
		NegotiatedSerializer: k8sscheme.Codecs.WithoutConversion(),
		VersionedAPIPath:     "/api",
	}

	err := confirmNodeUpgradeWithRESTClient(context.Background(), restClient)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send confirm request to MetaService API")
	assert.Contains(t, err.Error(), "request failed")
}

func newFakeRESTClient(roundTripper func(*http.Request) (*http.Response, error)) *restfake.RESTClient {
	return &restfake.RESTClient{
		Client:               restfake.CreateHTTPClient(roundTripper),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
		NegotiatedSerializer: k8sscheme.Codecs.WithoutConversion(),
		VersionedAPIPath:     "/api",
	}
}
