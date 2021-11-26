/*
Copyright 2021 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package podmanager

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// edgedMirrorClient implements the pod.MirrorClient interface.
// Edged does not have static pods and thus does not have mirror pods.
type edgedMirrorClient struct{}

// CreateMirrorPod is a no-op because edged does not have mirror pods.
func (emc *edgedMirrorClient) CreateMirrorPod(pod *v1.Pod) error {
	return nil
}

// DeleteMirrorPod is a no-op because edged does not have mirror pods.
func (emc *edgedMirrorClient) DeleteMirrorPod(podFullName string, uid *types.UID) (bool, error) {
	return false, nil
}
