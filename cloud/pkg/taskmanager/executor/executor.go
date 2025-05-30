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

package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/wrap"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
	"github.com/kubeedge/kubeedge/pkg/util/slices"
)

var (
	// nodeTaskExecutors is the map of node task executors.
	// The running executor will be in the map until it is
	// removed from the map after execution is completed.
	nodeTaskExecutors sync.Map

	ErrExecutorNotExists = errors.New("executor not exists")
)

// executorsKey returns the key of the node task executor.
// The key consists of the {resource_type}/{job_name}.
func executorsKey(resourceType, jobName string) string {
	return strings.Join([]string{resourceType, jobName}, "/")
}

// NewNodeTaskExecutor create an executor and add to nodeTaskExecutors.
// If one already exists in nodeTaskExecutors, use it.
func NewNodeTaskExecutor(ctx context.Context, job wrap.NodeJob, updateFun UpdateNodeTaskStatus,
) (*NodeTaskExecutor, bool, error) {
	logger := klog.FromContext(ctx).WithName("executor").
		WithValues("jobname", job.Name(), "jobtype", job.ResourceType())

	key := executorsKey(job.ResourceType(), job.Name())

	actual, loaded := nodeTaskExecutors.LoadOrStore(key, &NodeTaskExecutor{
		job:                  job,
		pool:                 NewPool(job.Concurrency()),
		messageLayer:         messagelayer.TaskManagerMessageLayer(),
		UpdateNodeTaskStatus: updateFun,
		logger:               logger,
	})
	executor, ok := actual.(*NodeTaskExecutor)
	if !ok {
		return nil, false,
			fmt.Errorf("failed to convert %s executor to NodeTaskExecutor, actual %T",
				key, executor)
	}
	return executor, loaded, nil
}

// GetExecutor returns the found executors from the nodeTaskExecutors,
// found by resource type and job name.
func GetExecutor(resourceType, jobName string) (*NodeTaskExecutor, error) {
	key := executorsKey(resourceType, jobName)
	actual, loaded := nodeTaskExecutors.Load(key)
	if !loaded {
		return nil, ErrExecutorNotExists
	}
	executor, ok := actual.(*NodeTaskExecutor)
	if !ok {
		return nil, fmt.Errorf("failed to convert %s executor to NodeTaskExecutor, actual %T",
			key, executor)
	}
	return executor, nil
}

// RemoveExecutor removes the executor from the nodeTaskExecutors,
// found by resource type and job name.
func RemoveExecutor(resourceType, jobName string) {
	nodeTaskExecutors.Delete(executorsKey(resourceType, jobName))
}

type NodeTaskExecutor struct {
	// job is the node job to be executed.
	job wrap.NodeJob
	// pool is the pool of concurrent resources.
	pool *Pool
	// interrupted indicates whether the executor is interrupted.
	interrupted atomic.Bool
	// messageLayer defines the message layer used to send edge nodes.
	messageLayer messagelayer.MessageLayer
	// UpdateNodeTaskStatus defines a function to update the status of the node task.
	UpdateNodeTaskStatus UpdateNodeTaskStatus
	// logger is the logger for the executor.
	logger logr.Logger
	// wg is the wait group for the executor. Used to wait for all node tasks to complete
	// before deleting the executor.
	wg sync.WaitGroup
}

type UpdateNodeTaskStatus func(ctx context.Context, job wrap.NodeJob, task wrap.NodeJobTask)

// Execute executes the node tasks. It uses a pool to control the number of concurrent executions of node tasks.
// The connectedNodes arg indicates the edge nodes that the current CloudCore is connected to. Only these nodes
// will execute tasks.
func (executor *NodeTaskExecutor) Execute(ctx context.Context, connectedNodes []string) {
	// All node tasks of the node job have been executed, delete the executor.
	defer RemoveExecutor(executor.job.ResourceType(), executor.job.Name())

	tasks := executor.job.Tasks()
	for i := range tasks {
		if executor.interrupted.Load() {
			return
		}
		task := tasks[i]
		if !task.CanExecute() {
			executor.logger.Info("the node does not meet the execution conditions, skip it",
				"nodename", task.NodeName())
			continue
		}

		executor.logger.V(2).Info("acquire a pool item ...")
		executor.pool.Acquire()
		executor.logger.V(2).Info("do node action", "nodename", task.NodeName())
		executor.wg.Add(1)

		if err := executor.executeTask(ctx, task, connectedNodes); err != nil {
			task.SetPhase(operationsv1alpha2.NodeTaskPhaseFailure, err.Error())
			executor.FinishTask()
		} else {
			if task.Phase() != operationsv1alpha2.NodeTaskPhaseInProgress {
				task.SetPhase(operationsv1alpha2.NodeTaskPhaseInProgress)
			}
		}
		executor.UpdateNodeTaskStatus(ctx, executor.job, task)
	}
	executor.wg.Wait()
}

// executeTask executes the node task. It sends a message to the edge node to execute the task.
func (executor *NodeTaskExecutor) executeTask(_ctx context.Context, task wrap.NodeJobTask, connectedNodes []string,
) error {
	if !slices.In(connectedNodes, task.NodeName()) {
		return fmt.Errorf("the node %s is not connected to the current cloudcore instance", task.NodeName())
	}
	msgres := taskmsg.Resource{
		APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
		ResourceType: executor.job.ResourceType(),
		JobName:      executor.job.Name(),
		NodeName:     task.NodeName(),
	}
	action, err := task.Action()
	if err != nil {
		return fmt.Errorf("failed to get node task action, err: %v", err)
	}
	msg := messagelayer.BuildNodeTaskRouter(msgres, action.Name).
		FillBody(executor.job.Spec())
	if err := executor.messageLayer.Send(*msg); err != nil {
		return fmt.Errorf("failed to send message to edge, err: %v", err)
	}
	return nil
}

// FinishTask when a node task has been completed, this function needs to be executed.
// Whether the node task is completed is sensed in the upstream.
func (executor *NodeTaskExecutor) FinishTask() {
	executor.logger.V(2).Info("release a pool item")
	executor.pool.Release()
	executor.wg.Done()
}

// Interrupt interrupts the executor
func (executor *NodeTaskExecutor) Interrupt() {
	executor.logger.V(2).Info("interrupt the executor")
	executor.interrupted.Store(true)
}
