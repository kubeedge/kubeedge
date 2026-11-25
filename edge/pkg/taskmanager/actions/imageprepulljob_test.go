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

package actions

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"

	edgecoreconfig "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	metaclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/pkg/image"
)

func TestImagePrePullJobCheckItems(t *testing.T) {
	ctx := context.TODO()
	specser := &cachedSpecSerializer{
		spec: &operationsv1alpha2.ImagePrePullJobSpec{
			ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
				CheckItems: []string{"cpu", "mem", "disk"},
			},
		},
	}
	h := imagePrePullJobActionHandler{
		logger: klog.Background(),
	}

	t.Run("check items failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(PreCheck, func([]string) error {
			return errors.New("test error")
		})

		resp := h.checkItems(ctx, "", "", specser)
		require.EqualError(t, resp.Error(), "test error")
	})

	t.Run("check items success", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(PreCheck, func([]string) error {
			return nil
		})

		resp := h.checkItems(ctx, "", "", specser)
		require.NoError(t, resp.Error())
	})
}

func TestImagePrePullJobPullImages(t *testing.T) {
	ctx := context.TODO()
	specser := &cachedSpecSerializer{
		spec: &operationsv1alpha2.ImagePrePullJobSpec{
			ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
				Images: []string{"image1", "image2"},
			},
		},
	}
	h := imagePrePullJobActionHandler{
		logger: klog.Background(),
	}

	imagert := &image.RuntimeImpl{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(options.GetEdgeCoreConfig, func() *edgecoreconfig.EdgeCoreConfig {
		return edgecoreconfig.NewDefaultEdgeCoreConfig()
	})
	patches.ApplyFunc(image.NewImageRuntime, func(_endpoint string, _timeout time.Duration,
	) (*image.RuntimeImpl, error) {
		return imagert, nil
	})
	patches.ApplyMethodFunc(reflect.TypeOf(imagert), "PullImage",
		func(_ctx context.Context,
			image string,
			_authConfig *runtimeapi.AuthConfig,
			_sandboxConfig *runtimeapi.PodSandboxConfig,
		) error {
			if image == "image2" {
				return errors.New("test error")
			}
			return nil
		})

	resp := h.pullImages(ctx, "", "", specser)
	require.ErrorContains(t, resp.Error(), pullImageFailureMessage)
	imagePrePullResp, ok := resp.(*imagePrePullJobActionResponse)
	assert.True(t, ok)
	assert.Equal(t, metav1.ConditionTrue, imagePrePullResp.imageStatus[0].Status)
	assert.Equal(t, metav1.ConditionFalse, imagePrePullResp.imageStatus[1].Status)
	assert.Equal(t, "test error", imagePrePullResp.imageStatus[1].Reason)
}

func TestImagePrePullJobPullImagesWithSecret(t *testing.T) {
	ctx := context.TODO()
	h := imagePrePullJobActionHandler{
		logger: klog.Background(),
	}
	newSpec := func(secret string) *operationsv1alpha2.ImagePrePullJobSpec {
		return &operationsv1alpha2.ImagePrePullJobSpec{
			ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
				Images:      []string{"registry.example.com/team/app:v1", "nginx:latest"},
				ImageSecret: secret,
			},
		}
	}

	imagert := &image.RuntimeImpl{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(options.GetEdgeCoreConfig, func() *edgecoreconfig.EdgeCoreConfig {
		return edgecoreconfig.NewDefaultEdgeCoreConfig()
	})
	patches.ApplyFunc(image.NewImageRuntime, func(_endpoint string, _timeout time.Duration,
	) (*image.RuntimeImpl, error) {
		return imagert, nil
	})
	pulled := map[string]*runtimeapi.AuthConfig{}
	patches.ApplyMethodFunc(reflect.TypeOf(imagert), "PullImage",
		func(_ctx context.Context,
			image string,
			authConfig *runtimeapi.AuthConfig,
			_sandboxConfig *runtimeapi.PodSandboxConfig,
		) error {
			pulled[image] = authConfig
			return nil
		})
	var getErr error
	patches.ApplyMethodFunc(reflect.TypeOf(metaclient.New().Secrets("default")), "Get",
		func(name string) (*corev1.Secret, error) {
			if getErr != nil {
				return nil, getErr
			}
			return &corev1.Secret{
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte(
					`{"auths":{"registry.example.com":{"username":"user","password":"pass"}}}`)},
			}, nil
		})

	t.Run("pull secret format is not correct", func(t *testing.T) {
		_, err := h.tryPullImage(ctx, newSpec("no-namespace"))
		require.EqualError(t, err, "pull secret format is not correct")
	})

	t.Run("get secret failed", func(t *testing.T) {
		getErr = errors.New("test error")
		defer func() { getErr = nil }()
		_, err := h.tryPullImage(ctx, newSpec("default/regcred"))
		require.ErrorContains(t, err, "failed to get secret default/regcred")
	})

	t.Run("credentials are used only for the matching registry", func(t *testing.T) {
		imageStatus, err := h.tryPullImage(ctx, newSpec("default/regcred"))
		require.NoError(t, err)
		require.Len(t, imageStatus, 2)
		auth := pulled["registry.example.com/team/app:v1"]
		require.NotNil(t, auth)
		assert.Equal(t, "user", auth.Username)
		assert.Equal(t, "pass", auth.Password)
		assert.Nil(t, pulled["nginx:latest"])
	})
}
