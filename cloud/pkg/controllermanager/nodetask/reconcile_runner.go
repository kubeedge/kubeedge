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
	"time"

	retry "github.com/avast/retry-go"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

const (
	NodeTaskReasonTimeout = "The node task has timed out"
)

// NodeJobType uses to constrain paradigm type of node jobs.
type NodeJobType interface {
	operationsv1alpha2.NodeUpgradeJob | operationsv1alpha2.ImagePrePullJob |
		operationsv1alpha2.ConfigUpdateJob
}

type ReconcileHandler[T NodeJobType] interface {
	//GetResource returns the resource name of k8s CRD.
	GetResource() string

	// GetJob returns the node job found by controller-runtime Client.
	// If not found, returns nil.
	GetJob(ctx context.Context, req controllerruntime.Request) (*T, error)

	// NoFinalizer returns whether the node job has no finalizer.
	NoFinalizer(job *T) bool

	// AddFinalizer adds the finalizer to the node job.
	AddFinalizer(ctx context.Context, job *T) error

	// RemoveFinalizer removes the finalizer from the node job.
	RemoveFinalizer(ctx context.Context, job *T) error

	// NotInitialized returns whether the node job is not initialized
	NotInitialized(job *T) bool

	// InitNodesStatus initializes nodes status for the node job.
	InitNodesStatus(ctx context.Context, job *T)

	// IsFinalPhase returns whether the node job is final phase.
	IsFinalPhase(job *T) bool

	// IsDeleted returns whether the node job is deleted.
	IsDeleted(job *T) bool

	// CalculateStatus calculates the node job phase through the all node tasks phases.
	CalculateStatus(ctx context.Context, job *T) bool

	// UpdateJobStatus updates the node job status by controller-runtime Client.
	UpdateJobStatus(ctx context.Context, job *T) error

	// CheckTimeout checks whather the node task has timed out. If so,
	// the node task needs to set the status to unknown, and update the resources.
	CheckTimeout(ctx context.Context, jobName string) error
}

const (
	UpdateStatusRetryAttempts = 3
	UpdateStatusRetryDelay    = 200 * time.Millisecond
)

// RunReconcile runs the common reconcile logic for the node job.
func RunReconcile[T NodeJobType](
	ctx context.Context,
	req controllerruntime.Request,
	handler ReconcileHandler[T],
) error {
	logger := klog.FromContext(ctx)
	job, err := handler.GetJob(ctx, req)
	if err != nil {
		return err
	}
	// The resource may no longer exist, in which case we stop processing.
	if job == nil {
		logger.V(2).Info("the node job is not found")
		return nil
	}

	if handler.NoFinalizer(job) {
		logger.V(2).Info("the node job has no finalizer, add finalizer")
		return handler.AddFinalizer(ctx, job)
	}

	if handler.IsDeleted(job) {
		ReleaseTimeoutJob[T](ctx, req.Name, handler.GetResource())
		return handler.RemoveFinalizer(ctx, job)
	}

	if handler.NotInitialized(job) {
		logger.V(2).Info("the node job is not initialized, initialize the status of node tasks")
		handler.InitNodesStatus(ctx, job)
		return handler.UpdateJobStatus(ctx, job)
	}

	// The final phase does not need to be calculated.
	if handler.IsFinalPhase(job) {
		logger.V(2).Info("the node job is final phase")
		// If job is in final phase, the the timeout job needs to be stopped.
		ReleaseTimeoutJob[T](ctx, req.Name, handler.GetResource())
		return nil
	}

	tj, loaded := GetOrCreateTimeoutJob(req.Name, handler.GetResource(), handler)
	if !loaded {
		go tj.Run(ctx)
	}

	// Retry update status error due to resourceVersion change.
	return retry.Do(
		func() error {
			job, err := handler.GetJob(ctx, req)
			if err != nil {
				return err
			}
			changed := handler.CalculateStatus(ctx, job)
			if changed {
				return handler.UpdateJobStatus(ctx, job)
			}
			return nil
		},
		retry.Delay(UpdateStatusRetryDelay),
		retry.Attempts(UpdateStatusRetryAttempts),
		retry.DelayType(retry.FixedDelay))
}
