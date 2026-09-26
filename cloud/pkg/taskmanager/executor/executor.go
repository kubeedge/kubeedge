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
	"time"

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
		ctx:                  ctx,
		taskTimeout:          job.Timeout(),
		running:              make(map[string]*runningTask),
		interrupt:            make(chan struct{}),
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

	// ctx controls the lifecycle of the executor. It is used by the asynchronous
	// timeout callbacks that outlive the Execute call.
	ctx context.Context
	// taskTimeout is the maximum duration to wait for a node to report a task
	// action before the task is marked as a timeout failure.
	taskTimeout time.Duration
	// mu guards running.
	mu sync.Mutex
	// running holds the in-flight node tasks keyed by node name, so that a task
	// can be completed exactly once, whether by the node report or by timeout.
	running map[string]*runningTask
	// interrupt is closed by Interrupt to unblock a running Execute.
	interrupt chan struct{}
	// interruptOnce guards closing the interrupt channel.
	interruptOnce sync.Once
}

// runningTask tracks an in-flight node task and its timeout timer.
type runningTask struct {
	task  wrap.NodeJobTask
	timer *time.Timer
	// gen is incremented every time the timeout is refreshed. A timeout callback
	// only fires when its generation still matches, which prevents a refresh that
	// races with an expiring timer from wrongly failing the task.
	gen int
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
			break
		}
		task := tasks[i]
		if !task.CanExecute() {
			executor.logger.Info("the node does not meet the execution conditions, skip it",
				"nodename", task.NodeName())
			continue
		}

		executor.logger.V(2).Info("acquire a pool item ...")
		if !executor.pool.AcquireWithContext(ctx, executor.interrupt) {
			break
		}
		executor.logger.V(2).Info("do node action", "nodename", task.NodeName())
		executor.wg.Add(1)

		// Mark the task in progress and start tracking it (including the timeout
		// countdown) before the message is sent, so that a completion report which
		// arrives while the message is still in flight is not missed.
		if task.Phase() != operationsv1alpha2.NodeTaskPhaseInProgress {
			task.SetPhase(operationsv1alpha2.NodeTaskPhaseInProgress)
		}
		executor.trackTask(task)

		if err := executor.executeTask(ctx, task, connectedNodes); err != nil {
			task.SetPhase(operationsv1alpha2.NodeTaskPhaseFailure, err.Error())
			executor.finishTask(task.NodeName())
		}
		executor.UpdateNodeTaskStatus(ctx, executor.job, task)
	}
	executor.wait(ctx)
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
func (executor *NodeTaskExecutor) FinishTask(nodeName string) {
	executor.logger.V(2).Info("release a pool item", "nodename", nodeName)
	executor.finishTask(nodeName)
}

// RefreshTask resets the timeout of the in-flight node task on nodeName. It is
// called when the node reports progress on a non-final action, so that a healthy
// but long multi-stage task is not wrongly failed.
func (executor *NodeTaskExecutor) RefreshTask(nodeName string) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	rt, ok := executor.running[nodeName]
	if !ok {
		return
	}
	rt.timer.Stop()
	rt.gen++
	rt.timer = executor.newTimeoutTimer(nodeName, rt.gen)
}

// trackTask registers an in-flight node task and starts its timeout timer.
func (executor *NodeTaskExecutor) trackTask(task wrap.NodeJobTask) {
	nodeName := task.NodeName()
	executor.mu.Lock()
	defer executor.mu.Unlock()
	rt := &runningTask{task: task}
	rt.timer = executor.newTimeoutTimer(nodeName, rt.gen)
	executor.running[nodeName] = rt
}

// newTimeoutTimer returns a timer that fails the node task on nodeName when it
// does not report within taskTimeout.
func (executor *NodeTaskExecutor) newTimeoutTimer(nodeName string, gen int) *time.Timer {
	return time.AfterFunc(executor.taskTimeout, func() {
		executor.timeoutTask(nodeName, gen)
	})
}

// finishTask removes and releases the in-flight node task on nodeName exactly
// once. It is a no-op if the task has already been completed.
func (executor *NodeTaskExecutor) finishTask(nodeName string) {
	executor.mu.Lock()
	rt, ok := executor.running[nodeName]
	if ok {
		delete(executor.running, nodeName)
		rt.timer.Stop()
	}
	executor.mu.Unlock()
	if !ok {
		return
	}
	executor.releaseSlot()
}

// timeoutTask fails the in-flight node task on nodeName when it has not reported
// within taskTimeout. The generation check ignores a stale timer that has been
// superseded by RefreshTask.
func (executor *NodeTaskExecutor) timeoutTask(nodeName string, gen int) {
	executor.mu.Lock()
	rt, ok := executor.running[nodeName]
	if !ok || rt.gen != gen {
		executor.mu.Unlock()
		return
	}
	delete(executor.running, nodeName)
	executor.mu.Unlock()

	executor.logger.Info("node task timed out waiting for completion, mark it failed",
		"nodename", nodeName, "timeout", executor.taskTimeout)
	rt.task.SetPhase(operationsv1alpha2.NodeTaskPhaseFailure,
		fmt.Sprintf("node %s did not report task completion within %s", nodeName, executor.taskTimeout))
	executor.UpdateNodeTaskStatus(executor.ctx, executor.job, rt.task)
	executor.releaseSlot()
}

// releaseSlot releases one concurrency slot and marks one tracked task done.
func (executor *NodeTaskExecutor) releaseSlot() {
	executor.pool.Release()
	executor.wg.Done()
}

// wait blocks until all in-flight node tasks complete, or returns early after
// draining them when the executor is interrupted or the context is canceled.
func (executor *NodeTaskExecutor) wait(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		executor.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-executor.interrupt:
		executor.drain()
		<-done
	case <-ctx.Done():
		executor.drain()
		<-done
	}
}

// drain stops all outstanding timeout timers and releases their concurrency
// slots, so that a wedged Execute can return promptly on interrupt or shutdown.
func (executor *NodeTaskExecutor) drain() {
	executor.mu.Lock()
	running := executor.running
	executor.running = make(map[string]*runningTask)
	executor.mu.Unlock()
	for _, rt := range running {
		rt.timer.Stop()
		executor.releaseSlot()
	}
}

// Interrupt interrupts the executor and unblocks a running Execute. Any
// in-flight node tasks are drained by Execute's wait.
func (executor *NodeTaskExecutor) Interrupt() {
	executor.logger.V(2).Info("interrupt the executor")
	executor.interrupted.Store(true)
	executor.interruptOnce.Do(func() {
		close(executor.interrupt)
	})
}
