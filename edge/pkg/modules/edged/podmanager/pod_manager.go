/*
Copyright 2015 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To create mini-kubelet for edge deployment scenario,
This file is derived from K8S Kubelet code with pruned structures and interfaces
and changed most of the realization.
1. Manager struct is dericed from basicManager struct in kubernetes/pkg/kubelet/pod/pod_manager.go
2. The methods are also pruned and modifed
*/

package podmanager

import (
	"sync"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/pod"
	"k8s.io/kubernetes/pkg/kubelet/pod/testing"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"
)

//Manager is derived from kubernetes/pkg/kubelet/pod/pod_manager.go
type Manager interface {
	pod.Manager
}

type podManager struct {
	lock          sync.RWMutex
	podByUID      map[types.UID]*v1.Pod
	podByFullName map[string]*v1.Pod
	*testing.MockManager
}

//NewPodManager creates new pod manager object
func NewPodManager() pod.Manager {
	pm := &podManager{
		MockManager: new(testing.MockManager),
	}
	pm.podByUID = make(map[types.UID]*v1.Pod)
	pm.podByFullName = make(map[string]*v1.Pod)
	return pm
}

func (pm *podManager) AddPod(pod *v1.Pod) {
	pm.UpdatePod(pod)
}

func (pm *podManager) UpdatePod(pod *v1.Pod) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.updatePodsInternal(pod)
}

func (pm *podManager) updatePodsInternal(pods ...*v1.Pod) {
	for _, pod := range pods {
		podFullName := container.GetPodFullName(pod)
		pm.podByUID[pod.UID] = pod
		pm.podByFullName[podFullName] = pod
	}
}

func podsMapToPods(UIDMap map[types.UID]*v1.Pod) []*v1.Pod {
	pods := make([]*v1.Pod, 0, len(UIDMap))
	for _, pod := range UIDMap {
		pods = append(pods, pod)
	}
	return pods
}

func (pm *podManager) GetPods() []*v1.Pod {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	return podsMapToPods(pm.podByUID)
}

func (pm *podManager) GetPodByUID(uid types.UID) (*v1.Pod, bool) {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	pod, ok := pm.podByUID[uid]
	return pod, ok
}

func (pm *podManager) GetPodByName(namespace, name string) (*v1.Pod, bool) {
	podFullName := container.BuildPodFullName(name, namespace)
	return pm.GetPodByFullName(podFullName)
}

func (pm *podManager) GetPodByFullName(podFullName string) (*v1.Pod, bool) {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	pod, ok := pm.podByFullName[podFullName]
	return pod, ok
}

func (pm *podManager) DeletePod(pod *v1.Pod) {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	podFullName := container.GetPodFullName(pod)
	delete(pm.podByUID, pod.UID)
	delete(pm.podByFullName, podFullName)
}

// GetUIDTranslations is part of the interface
// We don't have static pod, so don't have podToMirror and mirrorToPod
func (pm *podManager) GetUIDTranslations() (podToMirror map[kubetypes.ResolvedPodUID]kubetypes.MirrorPodUID,
	mirrorToPod map[kubetypes.MirrorPodUID]kubetypes.ResolvedPodUID) {
	return nil, nil
}

func (pm *podManager) TranslatePodUID(uid types.UID) kubetypes.ResolvedPodUID {
	return kubetypes.ResolvedPodUID(uid)
}

func (pm *podManager) GetMirrorPodByPod(*v1.Pod) (*v1.Pod, bool) {
	return nil, false
}
