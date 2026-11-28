/*
Copyright 2026 The KubeEdge Authors.

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

package manager

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/api/apis/operations/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha1/util"
)

func TestWorkersConcurrency(t *testing.T) {
	oldMachine := executorMachine
	defer func() { executorMachine = oldMachine }()

	executorMachine = &ExecutorMachine{
		executors:      map[string]*Executor{},
		downStreamChan: make(chan model.Message, 256),
	}

	timeout := uint32(1)
	const totalNodes = 32
	nodes := make([]v1alpha1.TaskStatus, totalNodes)
	for i := range nodes {
		nodes[i] = v1alpha1.TaskStatus{
			NodeName: fmt.Sprintf("node-%d", i),
			State:    api.TaskSuccessful,
		}
	}

	e := &Executor{
		task: util.TaskMessage{
			Name:           "task-1",
			Type:           "test",
			TimeOutSeconds: &timeout,
		},
		nodes: nodes,
		workers: workers{
			number: totalNodes,
			jobs:   make(map[string]int),
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < totalNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := e.workers.addJob(nodes[idx], idx, e)
			assert.NoError(t, err)
			_, _ = e.workers.endJob(nodes[idx].NodeName)
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = e.workers.jobCount()
		}
	}()

	wg.Wait()
	assert.Equal(t, 0, e.workers.jobCount())

	remaining := e.workers.shutdown()
	assert.Equal(t, 0, remaining)
	err := e.workers.addJob(nodes[0], 0, e)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped")
}

func TestHandleTimeOutJobNodeStateRace(t *testing.T) {
	timeout := uint32(5)
	e := &Executor{
		task: util.TaskMessage{
			Name:           "task-timeout-race",
			TimeOutSeconds: &timeout,
		},
		nodes: []v1alpha1.TaskStatus{
			{
				NodeName: "edge-node-1",
				State:    api.TaskChecking,
			},
		},
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.handelTimeOutJob(0)
	}()

	time.Sleep(20 * time.Millisecond)
	e.nodesMu.Lock()
	e.nodes[0].State = api.TaskSuccessful
	e.nodesMu.Unlock()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handelTimeOutJob did not exit after node state updated")
	}
}
