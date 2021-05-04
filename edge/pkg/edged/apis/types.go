package apis

import (
	"errors"
	"time"

	"github.com/docker/docker/api/types/container"
	api "k8s.io/kubernetes/pkg/apis/core"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
)

const (
	// NvidiaGPUStatusAnnotationKey is the key of the node annotation for GPU status
	NvidiaGPUStatusAnnotationKey = "huawei.com/gpu-status"
	// NvidiaGPUDecisionAnnotationKey is the key of the pod annotation for scheduler GPU decision
	NvidiaGPUDecisionAnnotationKey = "huawei.com/gpu-decision"
	// NvidiaGPUScalarResourceName is the device plugin resource name used for special handling
	NvidiaGPUScalarResourceName = "nvidia.com/gpu"
	// NvidiaGPUMaxUsage is the maximum possible usage of a GPU in millis
	NvidiaGPUMaxUsage = 1000
	// NvidiaGPUResource is the extend resource name
	NvidiaGPUResource = "alpha.kubernetes.io/nvidia-gpu"
	//StatusTag is to compare status of resources
	StatusTag = "StatusTag"
)

// Container defines container object
type Container struct {
	// ID of the container.
	ID string `json:"id,omitempty"`
	// Status of the container.
	Status  api.ContainerState `json:"status,omitempty"`
	StartAt time.Time          `json:"startat,omitempty"`
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

// ContainerConfig defines container configuration details
type ContainerConfig struct {
	Name       string
	Config     *container.Config
	HostConfig *container.HostConfig
}

// ContainerInspect is container inspect
type ContainerInspect struct {
	Status ContainerStatus `json:"Status,omitempty"`
}

// ContainerStatus represents the status of a container.
type ContainerStatus struct {
	api.ContainerStatus
	// Reference to the image in use. For most runtimes, this should be an
	// image ID
	ImageRef string `json:"image_ref,omitempty"`
	// Key-value pairs that may be used to scope and select individual resources.
	Labels map[string]string `json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Log path of container.
	LogPath      string `json:"log_path,omitempty"`
	RestartCount int32  `json:"restartCount"`
}

//error variables
var (
	ErrPodNotFound       = errors.New("PodNotFound")
	ErrContainerNotFound = errors.New("ContainerNotFound")
)

// RuntimeService is docker runtime service
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
