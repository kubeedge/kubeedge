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
	"time"

	"github.com/distribution/reference"
	digest "github.com/opencontainers/go-digest"
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
		return "", fmt.Errorf("image %s not found in local runtime", image)
	}

	repository, err := imageRepository(image)
	if err != nil {
		return "", fmt.Errorf("failed to parse image %s: %v", image, err)
	}
	for _, repoDigest := range resp.Image.RepoDigests {
		repoDigest = ConvToCRIImage(repoDigest)
		repo, dgst, err := parseRepositoryDigest(repoDigest)
		if err != nil {
			return "", fmt.Errorf("failed to parse repo digest %s for image %s: %v", repoDigest, image, err)
		}
		if repo == repository {
			return dgst, nil
		}
	}
	return "", fmt.Errorf("image digest for repository %s not found in local runtime", repository)
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

func NormalizeDigest(value string) (string, error) {
	dgst, err := digest.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid image digest %q: %v", value, err)
	}
	if err := dgst.Validate(); err != nil {
		return "", fmt.Errorf("invalid image digest %q: %v", value, err)
	}
	return dgst.String(), nil
}

func ImmutableImageRef(image, dgst string) (string, error) {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("failed to parse image %s: %v", image, err)
	}
	normalizedDigest, err := NormalizeDigest(dgst)
	if err != nil {
		return "", err
	}
	canonical, err := reference.WithDigest(reference.TrimNamed(named), digest.Digest(normalizedDigest))
	if err != nil {
		return "", fmt.Errorf("failed to build immutable image ref for %s: %v", image, err)
	}
	return canonical.String(), nil
}

func imageRepository(image string) (string, error) {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", err
	}
	return reference.TrimNamed(named).Name(), nil
}

func parseRepositoryDigest(repoDigest string) (string, string, error) {
	ref, err := reference.ParseNormalizedNamed(repoDigest)
	if err != nil {
		return "", "", err
	}
	digested, ok := ref.(reference.Digested)
	if !ok {
		return "", "", fmt.Errorf("reference %s does not include a digest", repoDigest)
	}
	normalizedDigest, err := NormalizeDigest(digested.Digest().String())
	if err != nil {
		return "", "", err
	}
	return reference.TrimNamed(ref).Name(), normalizedDigest, nil
}
