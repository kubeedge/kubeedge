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
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	criapitesting "k8s.io/cri-api/pkg/apis/testing"
)

const testImage = "docker.io/kubeedge/installation-package:v1.20.0"

// fakeImageServiceServer implements runtimeapi.ImageServiceServer.
// NewRemoteImageService validates the CRI endpoint by calling ImageFsInfo
// first, then falls back to ListImages — both must return valid responses.
type fakeImageServiceServer struct {
	runtimeapi.UnimplementedImageServiceServer
}

func (f *fakeImageServiceServer) ListImages(
	_ context.Context,
	_ *runtimeapi.ListImagesRequest,
) (*runtimeapi.ListImagesResponse, error) {
	return &runtimeapi.ListImagesResponse{}, nil
}

func (f *fakeImageServiceServer) ImageFsInfo(
	_ context.Context,
	_ *runtimeapi.ImageFsInfoRequest,
) (*runtimeapi.ImageFsInfoResponse, error) {
	return &runtimeapi.ImageFsInfoResponse{}, nil
}

func TestNewImageRuntime(t *testing.T) {
	t.Run("invalid endpoint scheme returns error", func(t *testing.T) {
		_, err := NewImageRuntime("invalid://test", time.Second)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to new remote image service")
	})

	t.Run("valid unix socket endpoint succeeds", func(t *testing.T) {
		socketPath := filepath.Join(t.TempDir(), "cri-test.sock")

		ln, err := net.Listen("unix", socketPath)
		require.NoError(t, err)

		srv := grpc.NewServer()
		runtimeapi.RegisterImageServiceServer(srv, &fakeImageServiceServer{})
		go func() { _ = srv.Serve(ln) }()
		defer srv.Stop()

		rt, err := NewImageRuntime("unix://"+socketPath, 5*time.Second)
		require.NoError(t, err)
		require.NotNil(t, rt)
	})
}

func TestPullImages(t *testing.T) {
	ctx := context.TODO()
	image := testImage

	t.Run("pull images success - image already present", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		fakeImgSvc.Images[image] = &runtimeapi.Image{
			RepoTags: []string{image},
		}
		imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}

		err := imgrt.PullImages(ctx, []string{image}, nil)
		require.NoError(t, err)
	})

	t.Run("pull images failure wraps error with image name", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		fakeImgSvc.InjectError("ImageStatus", errors.New("test image status error"))
		imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}

		err := imgrt.PullImages(ctx, []string{image}, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to pull image")
		require.ErrorContains(t, err, image)
	})
}

func TestPullImage(t *testing.T) {
	ctx := context.TODO()
	image := testImage

	t.Run("image status error returns error", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		fakeImgSvc.InjectError("ImageStatus", errors.New("test image status error"))
		imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}

		err := imgrt.PullImage(ctx, image, nil, nil)
		require.ErrorContains(t, err, "test image status error")
	})

	t.Run("image already present skips pull", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		fakeImgSvc.Images[image] = &runtimeapi.Image{
			RepoTags: []string{image},
		}
		imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}

		err := imgrt.PullImage(ctx, image, nil, nil)
		require.NoError(t, err)
	})

	t.Run("image not present pull succeeds", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}

		err := imgrt.PullImage(ctx, image, nil, nil)
		require.NoError(t, err)
	})

	t.Run("image not present pull fails returns error", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		fakeImgSvc.InjectError("PullImage", errors.New("test pull image error"))
		imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}

		err := imgrt.PullImage(ctx, image, nil, nil)
		require.ErrorContains(t, err, "test pull image error")
	})
}

func TestGetImageDigest(t *testing.T) {
	ctx := context.TODO()
	image := testImage

	t.Run("failed to get image status", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		fakeImgSvc.InjectError("ImageStatus", errors.New("test get image status error"))

		imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}
		_, err := imgrt.GetImageDigest(ctx, image)
		require.ErrorContains(t, err, "test get image status error")
	})

	t.Run("not found image status", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}
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

		imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}
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

		imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}
		digest, err := imgrt.GetImageDigest(ctx, image)
		require.NoError(t, err)
		require.Empty(t, digest)
	})
}

func TestConvToCRIImage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty input", input: "", expected: ""},
		{name: "malformed name - return as-is", input: "invalid@@image!!name", expected: "invalid@@image!!name"},
		{name: "docker.io with tag", input: "docker.io/kubeedge/installation-package:v1.20.0", expected: "docker.io/kubeedge/installation-package:v1.20.0"},
		{name: "docker.io default registry", input: "kubeedge/cloudcore:v1.21.0", expected: "docker.io/kubeedge/cloudcore:v1.21.0"},
		{name: "docker.io with digest", input: "docker.io/kubeedge/installation-package@sha256:abcd1234", expected: "docker.io/kubeedge/installation-package@sha256:abcd1234"},
		{name: "docker.io without tag", input: "docker.io/kubeedge/installation-package", expected: "docker.io/kubeedge/installation-package"},
		{name: "no registry or tag - default to docker.io", input: "kubeedge/installation-package", expected: "docker.io/kubeedge/installation-package"},
		{name: "private registry with tag", input: "registry.example.com/kubeedge/installation-package:v1.20.0", expected: "registry.example.com/kubeedge/installation-package:v1.20.0"},
		{name: "private registry without tag", input: "registry.example.com/kubeedge/installation-package", expected: "registry.example.com/kubeedge/installation-package"},
		{name: "internal registry no namespace", input: "internal-registry.net/cloudcore:v1.20.0", expected: "internal-registry.net/cloudcore:v1.20.0"},
		{name: "internal registry without tag", input: "internal-registry.net/installation-package", expected: "internal-registry.net/installation-package"},
		{name: "port registry no namespace", input: "registry.local:5000/cloudcore:v1.20.0", expected: "registry.local:5000/cloudcore:v1.20.0"},
		{name: "port registry with namespace", input: "registry.local:5000/kubeedge/installation-package:latest", expected: "registry.local:5000/kubeedge/installation-package:latest"},
		{name: "port registry no tag", input: "registry.local:5000/kubeedge/installation-package", expected: "registry.local:5000/kubeedge/installation-package"},
		{name: "port registry no namespace (alt)", input: "registry:5000/cloudcore:v1.20.0", expected: "registry:5000/cloudcore:v1.20.0"},
		{name: "port registry with namespace (alt)", input: "registry:5000/kubeedge/installation-package:latest", expected: "registry:5000/kubeedge/installation-package:latest"},
		{name: "port registry no tag (alt)", input: "registry:5000/kubeedge/installation-package", expected: "registry:5000/kubeedge/installation-package"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output := ConvToCRIImage(tc.input)
			require.Equal(t, tc.expected, output)
		})
	}
}
