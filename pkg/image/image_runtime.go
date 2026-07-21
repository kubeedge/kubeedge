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
	remote "k8s.io/cri-client/pkg"
	"k8s.io/klog/v2"
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
	logger := klog.Background()
	imgsvc, err := remote.NewRemoteImageService(endpoint, timeout, noop.NewTracerProvider(), &logger)
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

	// Confirm the requested tag exists in RepoTags.
	tagFound := false
	for _, tag := range resp.Image.RepoTags {
		if tag == image {
			tagFound = true
			break
		}
	}
	if !tagFound {
		return "", nil
	}

	// Extract repository name.
	ref, err := reference.ParseAnyReference(image)
	if err != nil {
		return "", fmt.Errorf("invalid image reference %s: %w", image, err)
	}
	named, ok := ref.(reference.Named)
	if !ok {
		return "", fmt.Errorf("invalid image reference %s: not a named reference", image)
	}
	repo := named.Name()

	// RepoTags and RepoDigests are independent fields with no guaranteed index
	// alignment. Iterate RepoDigests separately and match by repository name.
	for _, repoDigest := range resp.Image.RepoDigests {
		if !strings.Contains(repoDigest, "@") {
			continue
		}
		parsed, err := reference.ParseAnyReference(repoDigest)
		if err != nil {
			return "", fmt.Errorf("invalid digest for image %s: %w", image, err)
		}
		canonical, ok := parsed.(reference.Canonical)
		if !ok {
			continue
		}
		if canonical.Name() != repo {
			continue
		}
		return canonical.Digest().String(), nil
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
