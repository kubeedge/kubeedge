package dockertools

import (
	"io"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/flowcontrol"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
)

var _ kubecontainer.Runtime = &DockerManager{}

//APIVersion returns docker version
func (dm *DockerManager) APIVersion() (kubecontainer.Version, error) { return dm.Version() }

// GetContainerLogs returns logs of a specific container. By
// default, it returns a snapshot of the container log.
func (dm *DockerManager) GetContainerLogs(pod *v1.Pod, containerID kubecontainer.ContainerID, logOptions *v1.PodLogOptions, stdout, stderr io.Writer) (err error) {
	return nil
}

// GetPodContainerID returns a container in the pod with the given ID.
func (dm *DockerManager) GetPodContainerID(*kubecontainer.Pod) (kubecontainer.ContainerID, error) {
	return kubecontainer.ContainerID{}, nil
}

// GetNetNS returns the network namespace path for the given container
func (dm *DockerManager) GetNetNS(containerID kubecontainer.ContainerID) (string, error) {
	return "", nil
}

//GetPodStatus returns pod status
func (dm *DockerManager) GetPodStatus(uid types.UID, name, namespace string) (*kubecontainer.PodStatus, error) {
	return nil, nil
}

//KillPod ends the pod
func (dm *DockerManager) KillPod(pod *v1.Pod, runningPod kubecontainer.Pod, gracePeriodOverride *int64) error {
	return nil
}

//Status returns runtime status
func (dm *DockerManager) Status() (*kubecontainer.RuntimeStatus, error) { return nil, nil }

//SyncPod is to synchronise pods
func (dm *DockerManager) SyncPod(pod *v1.Pod, apiPodStatus v1.PodStatus, podStatus *kubecontainer.PodStatus, pullSecrets []v1.Secret, backOff *flowcontrol.Backoff) kubecontainer.PodSyncResult {
	return kubecontainer.PodSyncResult{}
}

//Type is string var to define typeof docker
func (dm *DockerManager) Type() string { return "" }

//UpdatePodCIDR to update pod CIDR
func (dm *DockerManager) UpdatePodCIDR(podCIDR string) error { return nil }
