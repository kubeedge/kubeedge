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

package util

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type mockImageService struct {
	ImageStatusFunc func(ctx context.Context, image *runtimeapi.ImageSpec, verbose bool) (*runtimeapi.ImageStatusResponse, error)
	PullImageFunc   func(ctx context.Context, image *runtimeapi.ImageSpec, auth *runtimeapi.AuthConfig, podSandboxConfig *runtimeapi.PodSandboxConfig) (string, error)
}

func (m *mockImageService) ListImages(ctx context.Context, filter *runtimeapi.ImageFilter) ([]*runtimeapi.Image, error) {
	return nil, nil
}

func (m *mockImageService) ImageStatus(ctx context.Context, image *runtimeapi.ImageSpec, verbose bool) (*runtimeapi.ImageStatusResponse, error) {
	if m.ImageStatusFunc != nil {
		return m.ImageStatusFunc(ctx, image, verbose)
	}
	return nil, nil
}

func (m *mockImageService) PullImage(ctx context.Context, image *runtimeapi.ImageSpec, auth *runtimeapi.AuthConfig, podSandboxConfig *runtimeapi.PodSandboxConfig) (string, error) {
	if m.PullImageFunc != nil {
		return m.PullImageFunc(ctx, image, auth, podSandboxConfig)
	}
	return "", nil
}

func (m *mockImageService) RemoveImage(ctx context.Context, image *runtimeapi.ImageSpec) error {
	return nil
}

func (m *mockImageService) ImageFsInfo(ctx context.Context) (*runtimeapi.ImageFsInfoResponse, error) {
	return nil, nil
}

type mockContainerRuntime struct {
	PullImagesFunc     func(images []string) error
	PullImageFunc      func(image string, authConfig *runtimeapi.AuthConfig, sandboxConfig *runtimeapi.PodSandboxConfig) error
	CopyResourcesFunc  func(edgeImage string, files map[string]string) error
	GetImageDigestFunc func(image string) (string, error)
}

func (m *mockContainerRuntime) PullImages(images []string) error {
	if m.PullImagesFunc != nil {
		return m.PullImagesFunc(images)
	}
	return nil
}

func (m *mockContainerRuntime) PullImage(image string, authConfig *runtimeapi.AuthConfig, sandboxConfig *runtimeapi.PodSandboxConfig) error {
	if m.PullImageFunc != nil {
		return m.PullImageFunc(image, authConfig, sandboxConfig)
	}
	return nil
}

func (m *mockContainerRuntime) CopyResources(edgeImage string, files map[string]string) error {
	if m.CopyResourcesFunc != nil {
		return m.CopyResourcesFunc(edgeImage, files)
	}
	return nil
}

func (m *mockContainerRuntime) GetImageDigest(image string) (string, error) {
	if m.GetImageDigestFunc != nil {
		return m.GetImageDigestFunc(image)
	}
	return "sha256:digest", nil
}

func TestConvertCRIImage(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		{
			name:     "Image with no path separator",
			image:    "nginx",
			expected: "docker.io/library/nginx",
		},
		{
			name:     "Image with one path separator",
			image:    "kubeedge/edgecore",
			expected: "docker.io/kubeedge/edgecore",
		},
		{
			name:     "Image with multiple path separators",
			image:    "docker.io/kubeedge/edgecore",
			expected: "docker.io/kubeedge/edgecore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertCRIImage(tt.image)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCopyResourcesCmd(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		check func(t *testing.T, cmd string)
	}{
		{
			name:  "No files",
			files: map[string]string{},
			check: func(t *testing.T, cmd string) {
				assert.Equal(t, "", cmd)
			},
		},
		{
			name: "Single file",
			files: map[string]string{
				"/path/in/container": "/path/on/host",
			},
			check: func(t *testing.T, cmd string) {
				assert.Equal(t, "cp /path/in/container /tmp/path/on/host", cmd)
			},
		},
		{
			name: "Multiple files",
			files: map[string]string{
				"/path1/in/container": "/path1/on/host",
				"/path2/in/container": "/path2/on/host",
			},
			check: func(t *testing.T, cmd string) {
				assert.Contains(t, cmd, "cp /path1/in/container /tmp/path1/on/host")
				assert.Contains(t, cmd, "cp /path2/in/container /tmp/path2/on/host")
				assert.Contains(t, cmd, " && ")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := copyResourcesCmd(tt.files)
			tt.check(t, cmd)
		})
	}
}

func TestPullImages(t *testing.T) {
	tests := []struct {
		name          string
		images        []string
		mockPullImage func(image string) error
		expectedError bool
	}{
		{
			name:   "Empty image list",
			images: []string{},
			mockPullImage: func(image string) error {
				return nil
			},
			expectedError: false,
		},
		{
			name:   "All images pull successfully",
			images: []string{"image1", "image2"},
			mockPullImage: func(image string) error {
				return nil
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime := &mockContainerRuntime{
				PullImageFunc: func(image string, authConfig *runtimeapi.AuthConfig, sandboxConfig *runtimeapi.PodSandboxConfig) error {
					return tt.mockPullImage(image)
				},
			}

			err := runtime.PullImages(tt.images)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCRIRuntime_PullImage(t *testing.T) {
	tests := []struct {
		name          string
		image         string
		setupMock     func(imageService *mockImageService)
		expectedError bool
	}{
		{
			name:  "Image status check fails",
			image: "nginx",
			setupMock: func(imageService *mockImageService) {
				imageService.ImageStatusFunc = func(ctx context.Context, image *runtimeapi.ImageSpec, verbose bool) (*runtimeapi.ImageStatusResponse, error) {
					return nil, errors.New("status check failed")
				}
			},
			expectedError: true,
		},
		{
			name:  "Image already exists",
			image: "nginx",
			setupMock: func(imageService *mockImageService) {
				imageService.ImageStatusFunc = func(ctx context.Context, image *runtimeapi.ImageSpec, verbose bool) (*runtimeapi.ImageStatusResponse, error) {
					return &runtimeapi.ImageStatusResponse{
						Image: &runtimeapi.Image{
							Id: "sha256:12345",
						},
					}, nil
				}
			},
			expectedError: false,
		},
		{
			name:  "Image doesn't exist and pull succeeds",
			image: "nginx",
			setupMock: func(imageService *mockImageService) {
				imageService.ImageStatusFunc = func(ctx context.Context, image *runtimeapi.ImageSpec, verbose bool) (*runtimeapi.ImageStatusResponse, error) {
					return nil, nil
				}
				imageService.PullImageFunc = func(ctx context.Context, image *runtimeapi.ImageSpec, auth *runtimeapi.AuthConfig, podSandboxConfig *runtimeapi.PodSandboxConfig) (string, error) {
					return "image-id", nil
				}
			},
			expectedError: false,
		},
		{
			name:  "Image doesn't exist and pull fails",
			image: "nginx",
			setupMock: func(imageService *mockImageService) {
				imageService.ImageStatusFunc = func(ctx context.Context, image *runtimeapi.ImageSpec, verbose bool) (*runtimeapi.ImageStatusResponse, error) {
					return nil, nil
				}
				imageService.PullImageFunc = func(ctx context.Context, image *runtimeapi.ImageSpec, auth *runtimeapi.AuthConfig, podSandboxConfig *runtimeapi.PodSandboxConfig) (string, error) {
					return "", errors.New("pull failed")
				}
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageService := &mockImageService{}
			tt.setupMock(imageService)

			runtime := &CRIRuntime{
				endpoint:            "unix:///var/run/dockershim.sock",
				cgroupDriver:        "cgroupfs",
				ImageManagerService: imageService,
				ctx:                 context.Background(),
			}

			err := runtime.PullImage(tt.image, nil, nil)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCRIRuntime_GetImageDigest(t *testing.T) {
	tests := []struct {
		name           string
		image          string
		setupMock      func(imageService *mockImageService)
		expectedDigest string
		expectedError  bool
	}{
		{
			name:  "Image status check fails",
			image: "nginx",
			setupMock: func(imageService *mockImageService) {
				imageService.ImageStatusFunc = func(ctx context.Context, image *runtimeapi.ImageSpec, verbose bool) (*runtimeapi.ImageStatusResponse, error) {
					return nil, errors.New("status check failed")
				}
			},
			expectedDigest: "",
			expectedError:  true,
		},
		{
			name:  "Successfully get image digest",
			image: "nginx",
			setupMock: func(imageService *mockImageService) {
				imageService.ImageStatusFunc = func(ctx context.Context, image *runtimeapi.ImageSpec, verbose bool) (*runtimeapi.ImageStatusResponse, error) {
					return &runtimeapi.ImageStatusResponse{
						Image: &runtimeapi.Image{
							Id: "sha256:12345",
							Spec: &runtimeapi.ImageSpec{
								Image: "docker.io/library/nginx@sha256:digest123",
							},
						},
					}, nil
				}
			},
			expectedDigest: "docker.io/library/nginx@sha256:digest123",
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageService := &mockImageService{}
			tt.setupMock(imageService)

			runtime := &CRIRuntime{
				endpoint:            "unix:///var/run/dockershim.sock",
				cgroupDriver:        "cgroupfs",
				ImageManagerService: imageService,
				ctx:                 context.Background(),
			}

			digest, err := runtime.GetImageDigest(tt.image)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedDigest, digest)
			}
		})
	}
}

func TestCRIRuntime_CopyResources(t *testing.T) {
	tests := []struct {
		name          string
		edgeImage     string
		files         map[string]string
		expectedError bool
	}{
		{
			name:      "Successful resource copy",
			edgeImage: "kubeedge/edgecore:latest",
			files: map[string]string{
				"/path/in/container/file1": "/path/on/host/file1",
				"/path/in/container/file2": "/path/on/host/file2",
			},
			expectedError: false,
		},
		{
			name:      "Resource copy fails",
			edgeImage: "kubeedge/edgecore:latest",
			files: map[string]string{
				"/path/in/container/file1": "/path/on/host/file1",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRuntime := &CRIRuntime{
				endpoint:     "unix:///var/run/dockershim.sock",
				cgroupDriver: "cgroupfs",
				ctx:          context.Background(),
			}

			patches := gomonkey.ApplyMethod(reflect.TypeOf(mockRuntime), "CopyResources",
				func(_ *CRIRuntime, edgeImage string, files map[string]string) error {
					if tt.expectedError {
						return errors.New("mocked error")
					}
					return nil
				})
			defer patches.Reset()

			err := mockRuntime.CopyResources(tt.edgeImage, tt.files)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewContainerRuntime(t *testing.T) {
	endpoint := "unix:///var/run/dockershim.sock"
	cgroupDriver := "cgroupfs"

	tests := []struct {
		name          string
		patchFunc     func(patches *gomonkey.Patches)
		expectedError bool
	}{
		{
			name: "Successfully create container runtime",
			patchFunc: func(patches *gomonkey.Patches) {
				patches.ApplyFunc(NewContainerRuntime,
					func(endpoint, cgroupDriver string) (ContainerRuntime, error) {
						return &mockContainerRuntime{}, nil
					})
			},
			expectedError: false,
		},
		{
			name: "Failed to create container runtime",
			patchFunc: func(patches *gomonkey.Patches) {
				patches.ApplyFunc(NewContainerRuntime,
					func(endpoint, cgroupDriver string) (ContainerRuntime, error) {
						return nil, errors.New("failed to create container runtime")
					})
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			tt.patchFunc(patches)

			runtime, err := NewContainerRuntime(endpoint, cgroupDriver)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, runtime)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, runtime)
			}
		})
	}
}
