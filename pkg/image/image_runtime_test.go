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

	godigest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	criapitesting "k8s.io/cri-api/pkg/apis/testing"
)

func TestGetImageDigest(t *testing.T) {
	ctx := context.TODO()
	image := "docker.io/kubeedge/installation-package:v1.20.0"
	validDigest := "sha256:e47afdf2746ad10ee76dd64289eae01895000327c0f23c5b498959eca6953695"
	validRepoDigest := "docker.io/kubeedge/installation-package@" + validDigest

	tests := []struct {
		name        string
		setup       func(*criapitesting.FakeImageService)
		wantDigest  string
		wantErr     bool
		errContains string
	}{
		{
			name: "ImageStatus error",
			setup: func(f *criapitesting.FakeImageService) {
				f.InjectError("ImageStatus", errors.New("test get image status error"))
			},
			wantErr:     true,
			errContains: "test get image status error",
		},
		{
			name:       "image not found",
			setup:      func(f *criapitesting.FakeImageService) {},
			wantDigest: "",
		},
		{
			name: "valid sha256 digest",
			setup: func(f *criapitesting.FakeImageService) {
				f.Images[image] = &runtimeapi.Image{
					RepoTags:    []string{image},
					RepoDigests: []string{validRepoDigest},
				}
			},
			wantDigest: validDigest,
		},
		{
			name: "tag not in RepoTags returns empty",
			setup: func(f *criapitesting.FakeImageService) {
				f.Images[image] = &runtimeapi.Image{
					RepoTags:    []string{"docker.io/kubeedge/installation-package:v1.19.0"},
					RepoDigests: []string{validRepoDigest},
				}
			},
			wantDigest: "",
		},
		{
			name: "RepoTags longer than RepoDigests no panic",
			setup: func(f *criapitesting.FakeImageService) {
				f.Images[image] = &runtimeapi.Image{
					RepoTags:    []string{image, "docker.io/kubeedge/installation-package:v1.19.0"},
					RepoDigests: []string{},
				}
			},
			wantDigest: "",
		},
		{
			name: "multiple tags and digests different ordering",
			setup: func(f *criapitesting.FakeImageService) {
				f.Images[image] = &runtimeapi.Image{
					RepoTags: []string{
						"docker.io/kubeedge/installation-package:v1.19.0",
						image,
					},
					RepoDigests: []string{
						"docker.io/kubeedge/other@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						validRepoDigest,
					},
				}
			},
			wantDigest: validDigest,
		},
		{
			name: "no matching repo in RepoDigests returns empty",
			setup: func(f *criapitesting.FakeImageService) {
				f.Images[image] = &runtimeapi.Image{
					RepoTags:    []string{image},
					RepoDigests: []string{"docker.io/kubeedge/other@" + validDigest},
				}
			},
			wantDigest: "",
		},
		{
			name: "empty digest after sha256 colon is invalid",
			setup: func(f *criapitesting.FakeImageService) {
				f.Images[image] = &runtimeapi.Image{
					RepoTags:    []string{image},
					RepoDigests: []string{"docker.io/kubeedge/installation-package@sha256:"},
				}
			},
			wantErr:     true,
			errContains: "invalid digest",
		},
		{
			name: "truncated digest is invalid",
			setup: func(f *criapitesting.FakeImageService) {
				f.Images[image] = &runtimeapi.Image{
					RepoTags:    []string{image},
					RepoDigests: []string{"docker.io/kubeedge/installation-package@sha256:123"},
				}
			},
			wantErr:     true,
			errContains: "invalid digest",
		},
		{
			name: "non-hex digest content is invalid",
			setup: func(f *criapitesting.FakeImageService) {
				f.Images[image] = &runtimeapi.Image{
					RepoTags:    []string{image},
					RepoDigests: []string{"docker.io/kubeedge/installation-package@sha256:not-hex"},
				}
			},
			wantErr:     true,
			errContains: "invalid digest",
		},
		{
			name: "digest missing at-sign separator is skipped",
			setup: func(f *criapitesting.FakeImageService) {
				f.Images[image] = &runtimeapi.Image{
					RepoTags:    []string{image},
					RepoDigests: []string{"docker.io/kubeedge/installation-package" + validDigest},
				}
			},
			wantDigest: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeImgSvc := criapitesting.NewFakeImageService()
			tc.setup(fakeImgSvc)
			imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}
			digest, err := imgrt.GetImageDigest(ctx, image)
			if tc.wantErr {
				require.ErrorContains(t, err, tc.errContains)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantDigest, digest)
		})
	}

	t.Run("returned digest parses cleanly", func(t *testing.T) {
		fakeImgSvc := criapitesting.NewFakeImageService()
		fakeImgSvc.Images[image] = &runtimeapi.Image{
			RepoTags:    []string{image},
			RepoDigests: []string{validRepoDigest},
		}
		imgrt := &RuntimeImpl{imgsvc: fakeImgSvc}
		digest, err := imgrt.GetImageDigest(ctx, image)
		require.NoError(t, err)
		d := godigest.Digest(digest)
		require.NoError(t, d.Validate())
	})
}

func TestConvToCRIImage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "malformed name - return as-is",
			input:    "invalid@@image!!name",
			expected: "invalid@@image!!name",
		},
		{
			name:     "docker.io with tag",
			input:    "docker.io/kubeedge/installation-package:v1.20.0",
			expected: "docker.io/kubeedge/installation-package:v1.20.0",
		},
		{
			name:     "docker.io default registry",
			input:    "kubeedge/cloudcore:v1.21.0",
			expected: "docker.io/kubeedge/cloudcore:v1.21.0",
		},
		{
			name:     "docker.io with digest",
			input:    "docker.io/kubeedge/installation-package@sha256:abcd1234",
			expected: "docker.io/kubeedge/installation-package@sha256:abcd1234",
		},
		{
			name:     "docker.io without tag",
			input:    "docker.io/kubeedge/installation-package",
			expected: "docker.io/kubeedge/installation-package",
		},
		{
			name:     "no registry or tag - default to docker.io",
			input:    "kubeedge/installation-package",
			expected: "docker.io/kubeedge/installation-package",
		},
		{
			name:     "private registry with tag",
			input:    "registry.example.com/kubeedge/installation-package:v1.20.0",
			expected: "registry.example.com/kubeedge/installation-package:v1.20.0",
		},
		{
			name:     "private registry without tag",
			input:    "registry.example.com/kubeedge/installation-package",
			expected: "registry.example.com/kubeedge/installation-package",
		},
		{
			name:     "internal registry no namespace",
			input:    "internal-registry.net/cloudcore:v1.20.0",
			expected: "internal-registry.net/cloudcore:v1.20.0",
		},
		{
			name:     "internal registry without tag",
			input:    "internal-registry.net/installation-package",
			expected: "internal-registry.net/installation-package",
		},
		{
			name:     "port registry no namespace",
			input:    "registry.local:5000/cloudcore:v1.20.0",
			expected: "registry.local:5000/cloudcore:v1.20.0",
		},
		{
			name:     "port registry with namespace",
			input:    "registry.local:5000/kubeedge/installation-package:latest",
			expected: "registry.local:5000/kubeedge/installation-package:latest",
		},
		{
			name:     "port registry no tag",
			input:    "registry.local:5000/kubeedge/installation-package",
			expected: "registry.local:5000/kubeedge/installation-package",
		},
		{
			name:     "port registry no namespace (alt)",
			input:    "registry:5000/cloudcore:v1.20.0",
			expected: "registry:5000/cloudcore:v1.20.0",
		},
		{
			name:     "port registry with namespace (alt)",
			input:    "registry:5000/kubeedge/installation-package:latest",
			expected: "registry:5000/kubeedge/installation-package:latest",
		},
		{
			name:     "port registry no tag (alt)",
			input:    "registry:5000/kubeedge/installation-package",
			expected: "registry:5000/kubeedge/installation-package",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output := ConvToCRIImage(tc.input)
			require.Equal(t, tc.expected, output)
		})
	}
}
