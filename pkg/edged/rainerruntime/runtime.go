package rainerruntime

import (
	"k8s.io/api/core/v1"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"

	"kubeedge/pkg/edged/containers"
)

type Runtime interface {
	containers.ContainerManager
	EnsureImageExists(pod *v1.Pod, secrets []v1.Secret) error
	Version() (kubecontainer.Version, error)
}
