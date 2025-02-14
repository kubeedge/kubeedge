/*
Copyright 2025 The KubeEdge Authors.

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

package downstream

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/nodes"
	"github.com/kubeedge/kubeedge/pkg/util/slices"
)

var nodeTaskExecutors sync.Map

type NodeTaskDownstream interface {
	JobName() string
	Tasks(ctx context.Context) []NodeTask
	SendTaskToEdge(ctx context.Context, task NodeTask) error
	HandleNodeTaskError(ctx context.Context, task NodeTask, err error)
}

type NodeTask interface {
	NodeName() string
	CanExecute() bool
}

func StartNodeTaskExecutor(ctx context.Context, ds NodeTaskDownstream, concurrency int) error {
	actual, loaded := nodeTaskExecutors.LoadOrStore(ds.JobName(),
		&NodeTaskExecutor{
			downstream: ds,
			pool:       make(chan struct{}, concurrency),
		})

	if loaded {
		return fmt.Errorf("node task executor %s already exists", ds.JobName())
	}
	executor, ok := actual.(*NodeTaskExecutor)
	if !ok {
		return fmt.Errorf("failed executor %s type, want NodeTaskExecutor, actual %T",
			ds.JobName(), executor)
	}
	sm, err := cloudhub.GetSessionManager()
	if err != nil {
		return fmt.Errorf("failed to get session manager, err: %v", err)
	}

	go executor.execute(ctx, nodes.GetManagedEdgeNodes(&sm.NodeSessions))
	return nil
}

func GetExecutor(taskName string) (*NodeTaskExecutor, error) {
	actual, loaded := nodeTaskExecutors.Load(taskName)
	if !loaded {
		return nil, fmt.Errorf("node task executor %s not exists", taskName)
	}
	executor, ok := actual.(*NodeTaskExecutor)
	if !ok {
		return nil, fmt.Errorf("failed executor %s type, want NodeTaskExecutor, actual %T",
			taskName, executor)
	}
	return executor, nil
}

type NodeTaskExecutor struct {
	downstream NodeTaskDownstream
	pool       chan struct{}
}

func (executor *NodeTaskExecutor) execute(ctx context.Context, connectedNodes []string) {
	defer nodeTaskExecutors.Delete(executor.downstream.JobName())

	logger := klog.FromContext(ctx)
	tasks := executor.downstream.Tasks(ctx)
	for i := range tasks {
		task := tasks[i]
		if !slices.In(connectedNodes, task.NodeName()) {
			logger.Info("the node has no connection to the current cloudcore instance, skip it",
				"nodename", task.NodeName())
			continue
		}
		if !task.CanExecute() {
			logger.Info("the node does not meet the execution conditions, skip it",
				"nodename", task.NodeName())
			continue
		}
		executor.pool <- struct{}{}
		if err := executor.downstream.SendTaskToEdge(ctx, task); err != nil {
			logger.Error(err, "failed to send node task to edge")
			executor.downstream.HandleNodeTaskError(ctx, task, err)
		}
	}
}

func (executor *NodeTaskExecutor) DoneOne() error {
	select {
	case <-executor.pool:
	default:
		return errors.New("no tasks are running")
	}
	return nil
}
