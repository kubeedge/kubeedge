/*
Copyright 2022 The KubeEdge Authors.

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

package worker

import (
	"sync"
)

type Factory struct {
	lock           sync.RWMutex
	workers        map[string]Worker
	startedWorkers map[string]bool
}

func NewWorkerFactory() *Factory {
	return &Factory{
		workers:        map[string]Worker{},
		startedWorkers: make(map[string]bool),
	}
}

// GetWorkerFor get worker for the resource type. For better
// performance, We do not add RLock for this function since
// this will be called by single goroutine `dispatchMessage`
// and no other goroutine will read or write the workers map.
func (f *Factory) GetWorkerFor(resourceType string) (Worker, bool) {
	worker, ok := f.workers[resourceType]
	return worker, ok
}

// Register register all resource process worker. this only be called
// when edgeController init phase.
func (f *Factory) Register(config WorkConfig) {
	f.lock.Lock()
	defer f.lock.Unlock()
	worker := NewWorker(config)
	f.workers[config.ResourceType] = worker
}

// Start initializes all requested workers.
func (f *Factory) Start() {
	f.lock.Lock()
	defer f.lock.Unlock()

	for resourceType, worker := range f.workers {
		if !f.startedWorkers[resourceType] {
			worker.Start()
			f.startedWorkers[resourceType] = true
		}
	}
}
