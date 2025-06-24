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
	"fmt"
	"strings"
	"time"
	"github.com/distribution/reference"
	"go.opentelemetry.io/otel/trace/noop"
	internalapi "k8s.io/cri-api/pkg/apis"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"
)

type Runtime interface {
	// PullImages pulls images. If authentication is required, currently pulled images
	// only support one authentication configuration.
	PullImages(ctx context.Context, images []string, authConfig *runtimeapi.AuthConfig) error
	// PullImage pulls the specified image.
	PullImage(ctx context.Context, image string, authConfig *runtimeapi.AuthConfig, sandboxConfig *runtimeapi.PodSandboxConfig) error
	// GetImageDigest returns the digest of the specified image.
	GetImageDigest(ctx context.Context, image string) (string, error)
}

type RuntimeImpl struct {
	endpoint string
	imgsvc   internalapi.ImageManagerService
}

// Check the RuntimeImpl implements the Runtime interface
var _ Runtime = (*RuntimeImpl)(nil)

func NewImageRuntime(endpoint string, timeout time.Duration) (*RuntimeImpl, error) {
	imgsvc, err := remote.NewRemoteImageService(endpoint, timeout, noop.NewTracerProvider())
	if err != nil {
		return nil, fmt.Errorf("failed to new remote image service, err: %v", err)
	}
	return &RuntimeImpl{
		imgsvc: imgsvc,
	}, nil
}

func (runtime *RuntimeImpl) PullImages(
	ctx context.Context,
	images []string,
	authConfig *runtimeapi.AuthConfig,
) error {
	for _, image := range images {
		if err := runtime.PullImage(ctx, image, authConfig, nil); err != nil {
			return fmt.Errorf("failed to pull image %s, err: %v", image, err)
		}
	}
	return nil
}

func (runtime *RuntimeImpl) GetImageDigest(ctx context.Context, image string) (string, error) {
	image = ConvToCRIImage(image)
	imageSpec := &runtimeapi.ImageSpec{Image: image}
	resp, err := runtime.imgsvc.ImageStatus(ctx, imageSpec, true)
	if err != nil {
		return "", err
	}
	if resp.Image == nil {
		return "", nil
	}
	for i := range resp.Image.RepoTags {
		tag := resp.Image.RepoTags[i]
		if tag == image {
			repoDigest := resp.Image.RepoDigests[i]
			digestIndex := strings.LastIndex(repoDigest, "@sha256:")
			return repoDigest[digestIndex+1:], nil
		}
	}
	return "", nil
}

func (runtime *RuntimeImpl) PullImage(
	ctx context.Context,
	image string,
	authConfig *runtimeapi.AuthConfig,
	sandboxConfig *runtimeapi.PodSandboxConfig,
) error {
	image = ConvToCRIImage(image)
	imageSpec := &runtimeapi.ImageSpec{Image: image}
	status, err := runtime.imgsvc.ImageStatus(ctx, imageSpec, true)
	if err != nil {
		return err
	}
	if status == nil || status.Image == nil {
		if _, err := runtime.imgsvc.PullImage(ctx, imageSpec, authConfig, sandboxConfig); err != nil {
			return err
		}
	}
	return nil
}

func ConvToCRIImage(image string) string {
	ref, err := reference.ParseAnyReference(image)
	if err != nil {
		return image
	}
	return ref.String()
}