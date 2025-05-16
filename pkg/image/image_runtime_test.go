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

package image

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	criapitesting "k8s.io/cri-api/pkg/apis/testing"
)

func TestGetImageDigest(t *testing.T) {
	ctx := context.TODO()
	image := "docker.io/kubeedge/installation-package:v1.20.0"

	t.Run("failed to get image status", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		fakeImgSvc.InjectError("ImageStatus", errors.New("test get image status error"))

		imgrt := &RuntimeImpl{
			imgsvc: fakeImgSvc,
		}
		_, err := imgrt.GetImageDigest(ctx, image)
		require.ErrorContains(t, err, "test get image status error")
	})

	t.Run("not found image status", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		imgrt := &RuntimeImpl{
			imgsvc: fakeImgSvc,
		}
		digest, err := imgrt.GetImageDigest(ctx, image)
		require.NoError(t, err)
		require.Equal(t, "", digest)
	})

	t.Run("match repo tags", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		fakeImgSvc.Images[image] = &runtimeapi.Image{
			RepoTags:    []string{image},
			RepoDigests: []string{"docker.io/kubeedge/installation-package@sha256:e47afdf2746ad10ee76dd64289eae01895000327c0f23c5b498959eca6953695"},
		}

		imgrt := &RuntimeImpl{
			imgsvc: fakeImgSvc,
		}
		digest, err := imgrt.GetImageDigest(ctx, image)
		require.NoError(t, err)
		require.Equal(t, "sha256:e47afdf2746ad10ee76dd64289eae01895000327c0f23c5b498959eca6953695", digest)
	})

	t.Run("not match repo tags", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		fakeImgSvc.Images[image] = &runtimeapi.Image{
			RepoTags:    []string{"kubeedge/installation-package:v1.20.0"},
			RepoDigests: []string{"kubeedge/installation-package@sha256:12345"},
		}

		imgrt := &RuntimeImpl{
			imgsvc: fakeImgSvc,
		}
		digest, err := imgrt.GetImageDigest(ctx, image)
		require.NoError(t, err)
		require.Empty(t, digest)
	})
}
