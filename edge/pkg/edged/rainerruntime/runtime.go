package rainerruntime

import (
	"k8s.io/api/core/v1"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/containers"
)

//Runtime is interface view run time status
type Runtime interface {
	containers.ContainerManager
	EnsureImageExists(pod *v1.Pod, secrets []v1.Secret) error
	Version() (kubecontainer.Version, error)
}
