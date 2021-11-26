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
1. The edgedManager struct is derived from the basicManager struct in kubernetes/pkg/kubelet/pod/pod_manager.go
2. The methods are also pruned and modifed since edged does not have static/mirror pods.
*/

package podmanager

import (
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/pod"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"
)

// edgedManager implements the pod.Manager interface.
//
// All fields are read-only and are updated calling SetPods, AddPod, UpdatePod, or DeletePod.
type edgedManager struct {
	// Protects all internal maps.
	lock sync.RWMutex

	// Pods indexed by UID.
	podByUID map[types.UID]*v1.Pod

	// Pods indexed by full name for easy access.
	podByFullName map[string]*v1.Pod

	// There are no static pods in edged, and thus no mirror pods.
	// This is here for interface compatibility.
	pod.MirrorClient
}

// NewEdgedPodManager returns a functional pod.Manager.
func NewEdgedPodManager() pod.Manager {
	em := &edgedManager{}
	em.MirrorClient = &edgedMirrorClient{}
	em.SetPods(nil)
	return em
}

// SetPods sets the internal pods based on new pods.
func (em *edgedManager) SetPods(newPods []*v1.Pod) {
	em.lock.Lock()
	defer em.lock.Unlock()

	em.podByUID = make(map[types.UID]*v1.Pod)
	em.podByFullName = make(map[string]*v1.Pod)

	em.updatePodsInternal(newPods...)
}

func (em *edgedManager) AddPod(pod *v1.Pod) {
	em.UpdatePod(pod)
}

func (em *edgedManager) UpdatePod(pod *v1.Pod) {
	em.lock.Lock()
	defer em.lock.Unlock()
	em.updatePodsInternal(pod)
}

// updatePodsInternal replaces the given pods in the current state of the
// manager, updating the various indices.  The caller is assumed to hold the
// lock.
func (em *edgedManager) updatePodsInternal(pods ...*v1.Pod) {
	for _, pod := range pods {
		podFullName := container.GetPodFullName(pod)
		em.podByUID[pod.UID] = pod
		em.podByFullName[podFullName] = pod
	}
}

func (em *edgedManager) DeletePod(pod *v1.Pod) {
	em.lock.Lock()
	defer em.lock.Unlock()
	podFullName := container.GetPodFullName(pod)
	delete(em.podByUID, pod.UID)
	delete(em.podByFullName, podFullName)
}

func (em *edgedManager) GetPods() []*v1.Pod {
	em.lock.RLock()
	defer em.lock.RUnlock()
	return podsMapToPods(em.podByUID)
}

// GetPodsAndMirrorPods returns a list of pods and nil because edged does not have mirror pods.
func (em *edgedManager) GetPodsAndMirrorPods() ([]*v1.Pod, []*v1.Pod) {
	em.lock.RLock()
	defer em.lock.RUnlock()
	pods := podsMapToPods(em.podByUID)

	return pods, nil
}

func (em *edgedManager) GetPodByUID(uid types.UID) (*v1.Pod, bool) {
	em.lock.RLock()
	defer em.lock.RUnlock()
	pod, ok := em.podByUID[uid]
	return pod, ok
}

func (em *edgedManager) GetPodByName(namespace, name string) (*v1.Pod, bool) {
	podFullName := container.BuildPodFullName(name, namespace)
	return em.GetPodByFullName(podFullName)
}

func (em *edgedManager) GetPodByFullName(podFullName string) (*v1.Pod, bool) {
	em.lock.RLock()
	defer em.lock.RUnlock()
	pod, ok := em.podByFullName[podFullName]
	return pod, ok
}

func (em *edgedManager) TranslatePodUID(uid types.UID) kubetypes.ResolvedPodUID {
	return kubetypes.ResolvedPodUID(uid)
}

// GetUIDTranslations is a no-op because edged does not have mirror pods.
func (em *edgedManager) GetUIDTranslations() (podToMirror map[kubetypes.ResolvedPodUID]kubetypes.MirrorPodUID,
	mirrorToPod map[kubetypes.MirrorPodUID]kubetypes.ResolvedPodUID) {
	return nil, nil
}

// GetOrphanedMirrorPods is a no-op because edged does not have mirror pods.
func (em *edgedManager) GetOrphanedMirrorPodNames() []string {
	return nil
}

// IsMirrorPodOf is a no-op because edged does not have mirror pods.
func (em *edgedManager) IsMirrorPodOf(mirrorPod, pod *v1.Pod) bool {
	return false
}

func podsMapToPods(UIDMap map[types.UID]*v1.Pod) []*v1.Pod {
	pods := make([]*v1.Pod, 0, len(UIDMap))
	for _, pod := range UIDMap {
		pods = append(pods, pod)
	}
	return pods
}

// GetMirrorPodByPod is a no-op because edged does not have mirror pods.
func (em *edgedManager) GetMirrorPodByPod(mirrodPod *v1.Pod) (*v1.Pod, bool) {
	return nil, false
}

// GetPodByMirrorPod is a no-op because edged does not have mirror pods.
func (em *edgedManager) GetPodByMirrorPod(mirrorPod *v1.Pod) (*v1.Pod, bool) {
	return nil, false
}
