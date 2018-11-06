package containers

import (
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/container"

	"kubeedge/beehive/pkg/common/log"
)

type sourceImpl struct{}

func (s sourceImpl) AddSource(source string) {
}

func (s sourceImpl) AllReady() bool {
	return true
}

// TODO: need consider EnvVar.ValueFrom
func ConvertEnvVersion(envs []v1.EnvVar) []container.EnvVar {
	var res []container.EnvVar
	for _, env := range envs {
		res = append(res, container.EnvVar{Name: env.Name, Value: env.Value})
	}
	return res
}

func GenerateEnvList(envs []v1.EnvVar) (result []string) {
	for _, env := range envs {
		result = append(result, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	return
}

func EnableHostUserNamespace(pod *v1.Pod) bool {
	if pod.Spec.Containers[0].SecurityContext != nil && *pod.Spec.Containers[0].SecurityContext.Privileged {
		return true
	}
	return false
}

func GenerateMountBindings(mounts []*container.Mount) []string {
	result := make([]string, 0, len(mounts))
	for _, m := range mounts {
		bind := fmt.Sprintf("%s:%s", m.HostPath, m.ContainerPath)
		var attrs []string
		if m.ReadOnly {
			attrs = append(attrs, "ro")
		}
		// Only request relabeling if the pod provides an SELinux context. If the pod
		// does not provide an SELinux context relabeling will label the volume with
		// the container's randomly allocated MCS label. This would restrict access
		// to the volume to the container which mounts it first.
		if m.SELinuxRelabel {
			attrs = append(attrs, "Z")
		}
		switch m.Propagation {
		case runtimeapi.MountPropagation_PROPAGATION_PRIVATE:
			// noop, private is default
		case runtimeapi.MountPropagation_PROPAGATION_BIDIRECTIONAL:
			attrs = append(attrs, "rshared")
		case runtimeapi.MountPropagation_PROPAGATION_HOST_TO_CONTAINER:
			attrs = append(attrs, "rslave")
		default:
			log.LOGGER.Warnf("unknown propagation mode for hostPath %q", m.HostPath)
			// Falls back to "private"
		}

		if len(attrs) > 0 {
			bind = fmt.Sprintf("%s:%s", bind, strings.Join(attrs, ","))
		}
		result = append(result, bind)
	}
	return result
}

func NewKubeContainerRuntime(cm ContainerManager) container.Runtime {
	return cm.(*containerManager)
}

type KubeSourcesReady struct{}

func (s *KubeSourcesReady) AllReady() bool {
	return true
}

// TODO: we didn't realized Run In container interface yet
func NewContainerRunner() container.ContainerCommandRunner {
	return &containerManager{}
}
