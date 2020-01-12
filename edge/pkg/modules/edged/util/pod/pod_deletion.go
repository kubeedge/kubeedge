package pod

import (
	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/status"
)

type podDeletionSafety struct{}

// TODO: add this function
// now assume pod can always be safety delete
func (p *podDeletionSafety) PodResourcesAreReclaimed(pod *v1.Pod, status v1.PodStatus) bool {
	return true
}

//NewPodDeleteSafety returns status of pod deletion safety
func NewPodDeleteSafety() status.PodDeletionSafetyProvider {
	return &podDeletionSafety{}
}
