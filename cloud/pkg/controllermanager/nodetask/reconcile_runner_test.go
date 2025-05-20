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
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

func TestRunReconcile(t *testing.T) {
	ctx := context.TODO()
	req := controllerruntime.Request{
		NamespacedName: types.NamespacedName{
			Name: "test-job",
		},
	}
	obj := &operationsv1alpha2.ImagePrePullJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-job",
		},
	}

	t.Run("job not found", func(t *testing.T) {
		handler := &fakeReconcileHandler{}
		err := RunReconcile(ctx, req, handler)
		assert.NoError(t, err)
		assert.Equal(t, 1, handler.called["GetJob"])
		assert.Equal(t, 0, handler.called["NoFinalizer"])
	})

	t.Run("job is no finalizer", func(t *testing.T) {
		handler := &fakeReconcileHandler{
			obj: obj,
		}
		err := RunReconcile(ctx, req, handler)
		assert.NoError(t, err)
		assert.Equal(t, 1, handler.called["NoFinalizer"])
		assert.Equal(t, 1, handler.called["AddFinalizer"])
		assert.Equal(t, 0, handler.called["CalculateStatus"])
		assert.Equal(t, handler.obj.Finalizers[0], operationsv1alpha2.FinalizerImagePrePullJob)
	})

	t.Run("job not init, init nodes status", func(t *testing.T) {
		handler := &fakeReconcileHandler{
			obj: obj,
		}
		err := RunReconcile(ctx, req, handler)
		assert.NoError(t, err)
		assert.Equal(t, 1, handler.called["NotInitialized"])
		assert.Equal(t, 1, handler.called["InitNodesStatus"])
		assert.Equal(t, 1, handler.called["UpdateJobStatus"])
		assert.Equal(t, 0, handler.called["CalculateStatus"])
		assert.Equal(t, operationsv1alpha2.JobPhaseInit, handler.obj.Status.Phase)
	})

	t.Run("job in progress, calculate status", func(t *testing.T) {
		handler := &fakeReconcileHandler{
			obj: obj,
		}
		err := RunReconcile(ctx, req, handler)
		assert.NoError(t, err)
		assert.Equal(t, 1, handler.called["CalculateStatus"])
		assert.Equal(t, 1, handler.called["UpdateJobStatus"])
	})

	t.Run("job is final phase", func(t *testing.T) {
		obj.Status.Phase = operationsv1alpha2.JobPhaseCompleted
		handler := &fakeReconcileHandler{
			obj: obj,
		}
		err := RunReconcile(ctx, req, handler)
		assert.NoError(t, err)
		assert.Equal(t, 0, handler.called["CalculateStatus"])
	})

	t.Run("job is deleted", func(t *testing.T) {
		obj.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		handler := &fakeReconcileHandler{
			obj: obj,
		}
		err := RunReconcile(ctx, req, handler)
		assert.NoError(t, err)
		assert.Equal(t, 0, handler.called["CalculateStatus"])
	})
}

type fakeReconcileHandler struct {
	// obj is the job object
	obj *operationsv1alpha2.ImagePrePullJob
	// called records the function call times
	called map[string]int
}

func (fakeReconcileHandler) GetResource() string {
	return operationsv1alpha2.ResourceImagePrePullJob
}

func (h *fakeReconcileHandler) GetJob(_ctx context.Context, _req controllerruntime.Request,
) (*operationsv1alpha2.ImagePrePullJob, error) {
	h.recordCalledFunc("GetJob")
	return h.obj, nil
}

func (h *fakeReconcileHandler) recordCalledFunc(name string) {
	if h.called == nil {
		h.called = make(map[string]int)
	}
	h.called[name]++
}

func (h *fakeReconcileHandler) NoFinalizer(job *operationsv1alpha2.ImagePrePullJob) bool {
	h.recordCalledFunc("NoFinalizer")
	return !controllerutil.ContainsFinalizer(job, operationsv1alpha2.FinalizerImagePrePullJob)
}

func (h *fakeReconcileHandler) AddFinalizer(ctx context.Context, job *operationsv1alpha2.ImagePrePullJob) error {
	h.recordCalledFunc("AddFinalizer")
	controllerutil.AddFinalizer(job, operationsv1alpha2.FinalizerImagePrePullJob)
	return nil
}

func (h *fakeReconcileHandler) RemoveFinalizer(ctx context.Context, job *operationsv1alpha2.ImagePrePullJob) error {
	h.recordCalledFunc("RemoveFinalizer")
	controllerutil.RemoveFinalizer(job, operationsv1alpha2.FinalizerImagePrePullJob)
	return nil
}

func (h *fakeReconcileHandler) NotInitialized(job *operationsv1alpha2.ImagePrePullJob) bool {
	h.recordCalledFunc("NotInitialized")
	return job.Status.Phase == ""
}

func (h *fakeReconcileHandler) InitNodesStatus(_ctx context.Context, job *operationsv1alpha2.ImagePrePullJob) {
	h.recordCalledFunc("InitNodesStatus")
	job.Status.Phase = operationsv1alpha2.JobPhaseInit
}

func (h *fakeReconcileHandler) IsFinalPhase(job *operationsv1alpha2.ImagePrePullJob) bool {
	h.recordCalledFunc("IsFinalPhase")
	return job.Status.Phase == operationsv1alpha2.JobPhaseCompleted ||
		job.Status.Phase == operationsv1alpha2.JobPhaseFailure
}

func (h *fakeReconcileHandler) IsDeleted(job *operationsv1alpha2.ImagePrePullJob) bool {
	h.recordCalledFunc("IsDeleted")
	return job.DeletionTimestamp != nil && !job.DeletionTimestamp.IsZero()
}

func (h *fakeReconcileHandler) CalculateStatus(_ctx context.Context, _job *operationsv1alpha2.ImagePrePullJob) bool {
	h.recordCalledFunc("CalculateStatus")
	return true
}

func (h *fakeReconcileHandler) UpdateJobStatus(ctx context.Context, _job *operationsv1alpha2.ImagePrePullJob) error {
	h.recordCalledFunc("UpdateJobStatus")
	return nil
}

func (h *fakeReconcileHandler) CheckTimeout(ctx context.Context, jobName string) error {
	h.recordCalledFunc("CheckTimeout")
	return nil
}
