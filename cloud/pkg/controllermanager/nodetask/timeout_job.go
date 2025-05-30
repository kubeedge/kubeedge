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

package nodetask

import (
	"context"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

var timeoutJobs sync.Map

// timeoutJobsKey used to generate the key of the timeout job sync map.
func timeoutJobsKey(jobName, resourceName string) string {
	return resourceName + "-" + jobName
}

// GetOrCreateTimeoutJob returns the timeout job if it exists, otherwise creates a new one.
func GetOrCreateTimeoutJob[T NodeJobType](jobName, resourceName string, handler ReconcileHandler[T],
) (*TimeoutJob[T], bool) {
	job, loaded := timeoutJobs.LoadOrStore(timeoutJobsKey(jobName, resourceName),
		NewTimeoutJob(jobName, handler))
	return job.(*TimeoutJob[T]), loaded
}

// ReleaseTimeoutJob releases the timeout job, removes it from the sync map, and stops it if it is running.
func ReleaseTimeoutJob[T NodeJobType](ctx context.Context, jobName, resourceName string) {
	job, loaded := timeoutJobs.LoadAndDelete(timeoutJobsKey(jobName, resourceName))
	if !loaded {
		return
	}
	if timeoutJob, ok := job.(*TimeoutJob[T]); ok {
		if !timeoutJob.IsStopped() {
			timeoutJob.Stop(ctx)
		}
	}
}

type TimeoutJob[T NodeJobType] struct {
	// nodeJobName is the name of the node job.
	nodeJobName string
	// ticker is used to trigger the timeout job.
	ticker *time.Ticker
	// handler an ReconcileHandler interface with a CheckTimeout function to check timeout.
	handler ReconcileHandler[T]
	// stopped indicates whether the timeout job is stopped.
	stopped bool
}

// NewTimeoutJob creates a new timeout job.
func NewTimeoutJob[T NodeJobType](jobName string, handler ReconcileHandler[T]) *TimeoutJob[T] {
	return &TimeoutJob[T]{
		nodeJobName: jobName,
		handler:     handler,
		ticker:      time.NewTicker(10 * time.Second),
	}
}

// Run runs the timeout job. It checks the timeout of the node job every 10 seconds(More appropriate frequency).
func (job *TimeoutJob[T]) Run(ctx context.Context) {
	logger := klog.FromContext(ctx)
	if job.stopped {
		logger.V(2).Info("timeout job is already stopped")
		return
	}

	for {
		select {
		case _, ok := <-job.ticker.C:
			if !ok { // channel closed
				logger.V(1).Info("ticker is closed")
				job.stopped = true
				return
			}
			if err := job.handler.CheckTimeout(ctx, job.nodeJobName); err != nil {
				logger.Error(err, "check timeout for job failed")
				job.ticker.Stop()
				return
			}
		case <-ctx.Done():
			logger.V(2).Info("timeout job is stopped by context")
			job.ticker.Stop()
			job.stopped = true
			return
		}
	}
}

// Stop stops the timeout job.
func (job *TimeoutJob[T]) Stop(ctx context.Context) {
	logger := klog.FromContext(ctx)
	logger.V(1).Info("stop the timeout job")
	job.ticker.Stop()
	job.stopped = true
}

// IsStopped returns whether the timeout job is stopped.
func (job TimeoutJob[T]) IsStopped() bool {
	return job.stopped
}
