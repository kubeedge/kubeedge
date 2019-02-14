package securitycontext

import (
	"strconv"

	"github.com/docker/docker/api/types/container"
	"k8s.io/api/core/v1"
)

//SimpleSecurityContextProvider is object ot define security context provider
type SimpleSecurityContextProvider struct{}

//NewSimpleSecurityContextProvider initialises and returns security context provider
func NewSimpleSecurityContextProvider() Provider {
	return SimpleSecurityContextProvider{}
}

//ModifyContainerConfig changes security context of container
func (s SimpleSecurityContextProvider) ModifyContainerConfig(pod *v1.Pod, config *container.Config) {
	securityContext := pod.Spec.Containers[0].SecurityContext
	if securityContext == nil {
		return
	}
	if securityContext.RunAsUser != nil {
		config.User = strconv.Itoa(int(*securityContext.RunAsUser))
	}
}

//ModifyHostConfig modifies security context of host
func (s SimpleSecurityContextProvider) ModifyHostConfig(pod *v1.Pod, hostConfig *container.HostConfig) {
	securityContext := pod.Spec.Containers[0].SecurityContext
	if securityContext != nil && securityContext.Privileged != nil {
		hostConfig.Privileged = *securityContext.Privileged
	}
}
