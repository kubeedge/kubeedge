/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.


@CHANGELOG
KubeEdge Authors: To create mini-kubelet for edge deployment scenario,
This file is derived from K8S Kubelet code with pruned structures and interfaces
and changed most of the realization.
Changes done are
1. For Runtime interface only ContainerManager and RuntimeVersion methods are considered.
2. Directly call docker client methods for container operations
*/

package cri

import (
	"time"

	"github.com/docker/docker/api/types/container"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
)

// Image defines basic information about a container image.
type Image struct {
	// ID of the image.
	ID string
	// Other names by which this image is known.
	RepoTags []string
	// Digests by which this image is known.
	RepoDigests []string
	// The size of the image in bytes.
	Size int64
}

//VersionResponse is object for versions
type VersionResponse struct {
	// Version of the kubelet runtime API.
	Version string `json:"version,omitempty"`

	// Name of the container runtime.
	RuntimeName string `json:"runtimeName,omitempty"`
	// Version of the container runtime. The string must be
	// semver-compatible.
	RuntimeVersion string `json:"runtimeVersion,omitempty"`
	// API version of the container runtime. The string must be
	// semver-compatible.
	RuntimeAPIVersion string `json:"runtimeApiVersion,omitempty"`
}

//constants to check status of container state
const (
	StatusUNKNOWN                              = kubecontainer.ContainerStateUnknown
	StatusCREATED                              = kubecontainer.ContainerStateCreated
	StatusRUNNING                              = kubecontainer.ContainerStateRunning
	StatusEXITED                               = kubecontainer.ContainerStateExited
	StatusSTOPPED kubecontainer.ContainerState = "stopped"
	StatusPAUSED  kubecontainer.ContainerState = "paused"
)

//Container defines container object
type Container struct {
	// ID of the container.
	ID string `json:"id,omitempty"`
	// Status of the container.
	Status  kubecontainer.ContainerState `json:"status,omitempty"`
	StartAt time.Time                    `json:"startat,omitempty"`
}

//ContainerInspect checks container status
type ContainerInspect struct {
	Status ContainerStatus `json:"Status,omitempty"`
}

// ContainerStatus represents the status of a container.
type ContainerStatus struct {
	kubecontainer.ContainerStatus
	// Reference to the image in use. For most runtimes, this should be an
	// image ID
	ImageRef string `json:"image_ref,omitempty"`
	// Key-value pairs that may be used to scope and select individual resources.
	Labels map[string]string `json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Log path of container.
	LogPath      string `json:"log_path,omitempty"`
	RestartCount int32  `json:"restartCount"`
}

// Device specifies a host device to mount into a container.
type Device struct {
	// Path of the device within the container.
	ContainerPath string `json:"container_path,omitempty"`
	// Path of the device on the host.
	HostPath string `json:"host_path,omitempty"`
	// Cgroups permissions of the device, candidates are one or more of
	// * r - allows container to read from the specified device.
	// * w - allows container to write to the specified device.
	// * m - allows container to create device files that do not yet exist.
	Permissions string `json:"permissions,omitempty"`
}

//ContainerConfig defines container configuration details
type ContainerConfig struct {
	Name       string
	Config     *container.Config
	HostConfig *container.HostConfig
}

//RuntimeService is interface for any run time service
type RuntimeService interface {
	Version() (kubecontainer.Version, error)
	CreateContainer(config *ContainerConfig) (string, error)
	StartContainer(containerID string) error
	StopContainer(containerID string, timeout uint32) error
	DeleteContainer(containerID kubecontainer.ContainerID) error
	ListContainers() ([]*Container, error)
	ContainerStatus(containerID string) (*ContainerStatus, error)
	InspectContainer(containerID string) (*ContainerInspect, error)
}

//constants for defining prefix in docker
const (
	DockerPrefix                = "docker://"
	DockerPullablePrefix        = "docker-pullable://"
	DockerImageIDPrefix         = DockerPrefix
	DockerPullableImageIDPrefix = DockerPullablePrefix
)
