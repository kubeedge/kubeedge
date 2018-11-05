package securitycontext

import (
	"github.com/docker/docker/api/types/container"
	"k8s.io/api/core/v1"
)

type SecurityContextProvider interface {
	ModifyContainerConfig(pod *v1.Pod, config *container.Config)
	ModifyHostConfig(pod *v1.Pod, hostConfig *container.HostConfig)
}
