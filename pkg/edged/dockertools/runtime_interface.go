package dockertools

import (
	"io"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/flowcontrol"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
)

var _ kubecontainer.Runtime = &DockerManager{}

func (dm *DockerManager) APIVersion() (kubecontainer.Version, error) { return dm.Version() }
func (dm *DockerManager) GetContainerLogs(pod *v1.Pod, containerID kubecontainer.ContainerID, logOptions *v1.PodLogOptions, stdout, stderr io.Writer) (err error) {
	return nil
}
func (dm *DockerManager) GetPodContainerID(*kubecontainer.Pod) (kubecontainer.ContainerID, error) {
	return kubecontainer.ContainerID{}, nil
}
func (dm *DockerManager) GetNetNS(containerID kubecontainer.ContainerID) (string, error) {
	return "", nil
}
func (dm *DockerManager) GetPodStatus(uid types.UID, name, namespace string) (*kubecontainer.PodStatus, error) {
	return nil, nil
}
func (dm *DockerManager) KillPod(pod *v1.Pod, runningPod kubecontainer.Pod, gracePeriodOverride *int64) error {
	return nil
}
func (dm *DockerManager) Status() (*kubecontainer.RuntimeStatus, error) { return nil, nil }
func (dm *DockerManager) SyncPod(pod *v1.Pod, apiPodStatus v1.PodStatus, podStatus *kubecontainer.PodStatus, pullSecrets []v1.Secret, backOff *flowcontrol.Backoff) kubecontainer.PodSyncResult {
	return kubecontainer.PodSyncResult{}
}
func (dm *DockerManager) Type() string                       { return "" }
func (dm *DockerManager) UpdatePodCIDR(podCIDR string) error { return nil }
