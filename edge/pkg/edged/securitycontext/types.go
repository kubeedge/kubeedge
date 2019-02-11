package securitycontext

import (
	"github.com/docker/docker/api/types/container"
	"k8s.io/api/core/v1"
)

//Provider is interface for security context modification
type Provider interface {
	ModifyContainerConfig(pod *v1.Pod, config *container.Config)
	ModifyHostConfig(pod *v1.Pod, hostConfig *container.HostConfig)
}
