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
	"testing"

	"github.com/stretchr/testify/assert"
	controllerruntime "sigs.k8s.io/controller-runtime"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

func TestRunReconcile(t *testing.T) {
	ctx := context.TODO()
	req := controllerruntime.Request{}

	t.Run("job not found", func(t *testing.T) {
		handler := &fakeReconcileHandler{}
		err := RunReconcile(ctx, req, handler)
		assert.NoError(t, err)
		assert.False(t, handler.initStatus)
		assert.False(t, handler.calculateStatus)
		assert.False(t, handler.updateStatus)
	})

	t.Run("job is final phase", func(t *testing.T) {
		handler := &fakeReconcileHandler{
			obj: &operationsv1alpha2.ImagePrePullJob{
				Status: operationsv1alpha2.ImagePrePullJobStatus{
					Phase: operationsv1alpha2.JobPhaseCompleted,
				},
			},
		}
		err := RunReconcile(ctx, req, handler)
		assert.NoError(t, err)
		assert.False(t, handler.initStatus)
		assert.False(t, handler.calculateStatus)
		assert.False(t, handler.updateStatus)
	})

	t.Run("job not init, init nodes status", func(t *testing.T) {
		handler := &fakeReconcileHandler{
			obj: &operationsv1alpha2.ImagePrePullJob{},
		}
		err := RunReconcile(ctx, req, handler)
		assert.NoError(t, err)
		assert.True(t, handler.initStatus)
		assert.False(t, handler.calculateStatus)
		assert.True(t, handler.updateStatus)
	})

	t.Run("job has initialized, calculate status", func(t *testing.T) {
		handler := &fakeReconcileHandler{
			obj: &operationsv1alpha2.ImagePrePullJob{
				Status: operationsv1alpha2.ImagePrePullJobStatus{
					Phase: operationsv1alpha2.JobPhaseInProgress,
				},
			},
		}
		err := RunReconcile(ctx, req, handler)
		assert.NoError(t, err)
		assert.False(t, handler.initStatus)
		assert.True(t, handler.calculateStatus)
		assert.True(t, handler.updateStatus)
	})
}

type fakeReconcileHandler struct {
	obj             *operationsv1alpha2.ImagePrePullJob
	initStatus      bool
	calculateStatus bool
	updateStatus    bool
}

func (h *fakeReconcileHandler) GetJob(_ctx context.Context, _req controllerruntime.Request,
) (*operationsv1alpha2.ImagePrePullJob, error) {
	return h.obj, nil
}

func (h *fakeReconcileHandler) NotInitialized(job *operationsv1alpha2.ImagePrePullJob) bool {
	return job.Status.Phase == ""
}

func (h *fakeReconcileHandler) IsFinalPhase(job *operationsv1alpha2.ImagePrePullJob) bool {
	return job.Status.Phase == operationsv1alpha2.JobPhaseCompleted ||
		job.Status.Phase == operationsv1alpha2.JobPhaseFailure
}

func (h *fakeReconcileHandler) InitNodesStatus(_ctx context.Context, _job *operationsv1alpha2.ImagePrePullJob) {
	h.initStatus = true
}

func (h *fakeReconcileHandler) CalculateStatus(_ctx context.Context, _job *operationsv1alpha2.ImagePrePullJob) bool {
	h.calculateStatus = true
	return true
}

func (h *fakeReconcileHandler) UpdateJobStatus(ctx context.Context, _job *operationsv1alpha2.ImagePrePullJob) error {
	h.updateStatus = true
	return nil
}
