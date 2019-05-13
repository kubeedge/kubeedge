/*
Copyright 2019 The KubeEdge Authors.

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

package handlerstub

import (
	"sync"

	"github.com/kubeedge/kubeedge/tests/stubs/common/types"
)

// NewPodManager creates pod manger
func NewPodManager() (*PodManager, error) {
	pm := &PodManager{}
	return pm, nil
}

// PodManager is a manager watch pod change event
type PodManager struct {
	// pods map
	pods sync.Map
}

// AddPod adds pod in cache
func (pm *PodManager) AddPod(k string, v types.FakePod) {
	pm.pods.Store(k, v)
}

// DeletePod deletes pod in cache
func (pm *PodManager) DeletePod(k string) {
	pm.pods.Delete(k)
}

// ListPods lists all pods in cache
func (pm *PodManager) ListPods() []types.FakePod {
	pods := make([]types.FakePod, 0)
	pm.pods.Range(func(k, v interface{}) bool {
		pods = append(pods, v.(types.FakePod))
		return true
	})
	return pods
}
