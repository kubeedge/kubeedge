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
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

func TestImagePrePullJobGetJob(t *testing.T) {
	ctx := context.TODO()
	cli := fakeImagePrePullJobClient(&operationsv1alpha2.ImagePrePullJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job"},
	})
	t.Run("not found", func(t *testing.T) {
		handler := NewImagePrePullJobReconcileHandler(cli, nil)
		obj, err := handler.GetJob(ctx, controllerruntime.Request{
			NamespacedName: client.ObjectKey{Name: "not-found"},
		})
		assert.NoError(t, err)
		assert.Nil(t, obj)
	})

	t.Run("get job successful", func(t *testing.T) {
		handler := NewImagePrePullJobReconcileHandler(cli, nil)
		obj, err := handler.GetJob(ctx, controllerruntime.Request{
			NamespacedName: client.ObjectKey{Name: "test-job"},
		})
		assert.NoError(t, err)
		assert.NotNil(t, obj)
	})
}

func TestImagePrePullJobFinalizer(t *testing.T) {
	ctx := context.TODO()
	cli := fakeImagePrePullJobClient(&operationsv1alpha2.ImagePrePullJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-job",
		},
	})
	handler := NewImagePrePullJobReconcileHandler(cli, nil)

	var found operationsv1alpha2.ImagePrePullJob
	var err error
	err = cli.Get(ctx, client.ObjectKey{Name: "test-job"}, &found)
	require.NoError(t, err)
	assert.True(t, handler.NoFinalizer(&found))

	err = handler.AddFinalizer(ctx, &found)
	require.NoError(t, err)
	err = cli.Get(ctx, client.ObjectKey{Name: "test-job"}, &found)
	require.NoError(t, err)
	assert.False(t, handler.NoFinalizer(&found))

	err = handler.RemoveFinalizer(ctx, &found)
	require.NoError(t, err)
	err = cli.Get(ctx, client.ObjectKey{Name: "test-job"}, &found)
	require.NoError(t, err)
	assert.True(t, handler.NoFinalizer(&found))
}

func TestImagePrePullJobNotInitialized(t *testing.T) {
	cases := []struct {
		name string
		obj  *operationsv1alpha2.ImagePrePullJob
		want bool
	}{
		{
			name: "phase is empty",
			obj:  &operationsv1alpha2.ImagePrePullJob{},
			want: true,
		},
		{
			name: "phase not emnpty",
			obj: &operationsv1alpha2.ImagePrePullJob{
				Status: operationsv1alpha2.ImagePrePullJobStatus{
					Phase: operationsv1alpha2.JobPhaseInit,
				},
			},
			want: false,
		},
	}

	handler := NewImagePrePullJobReconcileHandler(nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, handler.NotInitialized(c.obj))
		})
	}
}

func TestImagePrePullJobInitNodesStatus(t *testing.T) {
	ctx := context.TODO()

	t.Run("verify node define failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(VerifyNodeDefine,
			func(ctx context.Context,
				che cache.Cache,
				nodeNames []string,
				nodeSelector *metav1.LabelSelector,
			) (res []NodeVerificationResult, err error) {
				return nil, errors.New("failed to verify node define")
			})

		handler := NewImagePrePullJobReconcileHandler(nil, nil)
		job := &operationsv1alpha2.ImagePrePullJob{}
		handler.InitNodesStatus(ctx, job)
		assert.Equal(t, operationsv1alpha2.JobPhaseFailure, job.Status.Phase)
		assert.Equal(t, "failed to verify node define", job.Status.Reason)
	})

	t.Run("init nodes status successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(VerifyNodeDefine,
			func(ctx context.Context,
				che cache.Cache,
				nodeNames []string,
				nodeSelector *metav1.LabelSelector,
			) (res []NodeVerificationResult, err error) {
				return []NodeVerificationResult{
					{NodeName: "node1"},
					{NodeName: "node2", ErrorMessage: "failed to init node2"},
				}, nil
			})

		handler := NewImagePrePullJobReconcileHandler(nil, nil)
		job := &operationsv1alpha2.ImagePrePullJob{}
		handler.InitNodesStatus(ctx, job)
		assert.Equal(t, operationsv1alpha2.JobPhaseInit, job.Status.Phase)
		assert.Len(t, job.Status.NodeStatus, 2)

		assert.Equal(t, "node1", job.Status.NodeStatus[0].NodeName)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhasePending, job.Status.NodeStatus[0].Phase)
		assert.Empty(t, job.Status.NodeStatus[0].Reason)

		assert.Equal(t, "node2", job.Status.NodeStatus[1].NodeName)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhaseFailure, job.Status.NodeStatus[1].Phase)
		assert.Equal(t, "failed to init node2", job.Status.NodeStatus[1].Reason)
	})
}

func TestImagePrePullJobIsFinalPhase(t *testing.T) {
	cases := []struct {
		name string
		obj  *operationsv1alpha2.ImagePrePullJob
		want bool
	}{
		{
			name: "init is not a final phase",
			obj: &operationsv1alpha2.ImagePrePullJob{
				Status: operationsv1alpha2.ImagePrePullJobStatus{
					Phase: operationsv1alpha2.JobPhaseInProgress,
				},
			},
			want: false,
		},
		{
			name: "failure is a final phase",
			obj: &operationsv1alpha2.ImagePrePullJob{
				Status: operationsv1alpha2.ImagePrePullJobStatus{
					Phase: operationsv1alpha2.JobPhaseFailure,
				},
			},
			want: true,
		},
		{
			name: "complated is a final phase",
			obj: &operationsv1alpha2.ImagePrePullJob{
				Status: operationsv1alpha2.ImagePrePullJobStatus{
					Phase: operationsv1alpha2.JobPhaseCompleted,
				},
			},
			want: true,
		},
	}

	handler := NewImagePrePullJobReconcileHandler(nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, handler.IsFinalPhase(c.obj))
		})
	}
}

func TestImagePrePullJobIsDeleted(t *testing.T) {
	cases := []struct {
		name string
		obj  *operationsv1alpha2.ImagePrePullJob
		want bool
	}{
		{
			name: "not deleted",
			obj: &operationsv1alpha2.ImagePrePullJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-job",
				},
			},
			want: false,
		},
		{
			name: "deleted",
			obj: &operationsv1alpha2.ImagePrePullJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-job",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
			},
			want: true,
		},
	}
	handler := NewImagePrePullJobReconcileHandler(nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, handler.IsDeleted(c.obj))
		})
	}
}

func TestImagePrePullJobCalculateStatus(t *testing.T) {
	cases := []struct {
		name        string
		obj         *operationsv1alpha2.ImagePrePullJob
		wantChanged bool
		wantPhase   operationsv1alpha2.JobPhase
	}{
		{
			name: "some node tasks are in progress",
			obj: &operationsv1alpha2.ImagePrePullJob{
				Status: operationsv1alpha2.ImagePrePullJobStatus{
					Phase: operationsv1alpha2.JobPhaseInProgress,
					NodeStatus: []operationsv1alpha2.ImagePrePullNodeTaskStatus{
						{
							NodeName: "node1",
							Phase:    operationsv1alpha2.NodeTaskPhaseInProgress,
						},
						{
							NodeName: "node2",
							Phase:    operationsv1alpha2.NodeTaskPhaseSuccessful,
						},
						{
							NodeName: "node3",
							Phase:    operationsv1alpha2.NodeTaskPhaseFailure,
						},
					},
				},
			},
			wantChanged: false,
			wantPhase:   operationsv1alpha2.JobPhaseInProgress,
		},
		{
			name: "most node task are failure",
			obj: &operationsv1alpha2.ImagePrePullJob{
				Spec: operationsv1alpha2.ImagePrePullJobSpec{
					ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
						FailureTolerate: "0.5",
					},
				},
				Status: operationsv1alpha2.ImagePrePullJobStatus{
					Phase: operationsv1alpha2.JobPhaseInProgress,
					NodeStatus: []operationsv1alpha2.ImagePrePullNodeTaskStatus{
						{
							NodeName: "node1",
							Phase:    operationsv1alpha2.NodeTaskPhaseUnknown,
						},
						{
							NodeName: "node2",
							Phase:    operationsv1alpha2.NodeTaskPhaseSuccessful,
						},
						{
							NodeName: "node3",
							Phase:    operationsv1alpha2.NodeTaskPhaseFailure,
						},
					},
				},
			},
			wantChanged: true,
			wantPhase:   operationsv1alpha2.JobPhaseFailure,
		},
		{
			name: "most node task are successful",
			obj: &operationsv1alpha2.ImagePrePullJob{
				Spec: operationsv1alpha2.ImagePrePullJobSpec{
					ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
						FailureTolerate: "0.5",
					},
				},
				Status: operationsv1alpha2.ImagePrePullJobStatus{
					Phase: operationsv1alpha2.JobPhaseInProgress,
					NodeStatus: []operationsv1alpha2.ImagePrePullNodeTaskStatus{
						{
							NodeName: "node1",
							Phase:    operationsv1alpha2.NodeTaskPhaseSuccessful,
						},
						{
							NodeName: "node2",
							Phase:    operationsv1alpha2.NodeTaskPhaseSuccessful,
						},
						{
							NodeName: "node3",
							Phase:    operationsv1alpha2.NodeTaskPhaseFailure,
						},
					},
				},
			},
			wantChanged: true,
			wantPhase:   operationsv1alpha2.JobPhaseCompleted,
		},
	}

	ctx := context.TODO()
	handler := NewImagePrePullJobReconcileHandler(nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			changed := handler.CalculateStatus(ctx, c.obj)
			assert.Equal(t, c.wantChanged, changed)
			assert.Equal(t, c.wantPhase, c.obj.Status.Phase)
		})
	}
}

func TestImagePrePullJobUpdateJobStatus(t *testing.T) {
	ctx := context.TODO()
	job := &operationsv1alpha2.ImagePrePullJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job"},
	}
	cli := fakeImagePrePullJobClient(job)
	handler := NewImagePrePullJobReconcileHandler(cli, nil)
	job.Status.Phase = operationsv1alpha2.JobPhaseInProgress
	err := handler.UpdateJobStatus(ctx, job)
	assert.NoError(t, err)
}

func TestImagePrePullJobCheckTimeout(t *testing.T) {
	ctx := context.TODO()

	t.Run("job is not in progress", func(t *testing.T) {
		var updateJobStatusCalled bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*ImagePrePullJobReconcileHandler)(nil)), "GetJob",
			func(_ctx context.Context, _req controllerruntime.Request) (*operationsv1alpha2.ImagePrePullJob, error) {
				return &operationsv1alpha2.ImagePrePullJob{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-job",
					},
					Status: operationsv1alpha2.ImagePrePullJobStatus{
						Phase: operationsv1alpha2.JobPhaseInit,
					},
				}, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*ImagePrePullJobReconcileHandler)(nil)), "UpdateJobStatus",
			func(_ctx context.Context, _job *operationsv1alpha2.ImagePrePullJob) error {
				updateJobStatusCalled = true
				return nil
			})

		handler := &ImagePrePullJobReconcileHandler{}
		err := handler.CheckTimeout(ctx, "test-job")
		require.NoError(t, err)
		assert.False(t, updateJobStatusCalled)
	})

	t.Run("timeout seconds is not set", func(t *testing.T) {
		var updateJobStatusCalled bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*ImagePrePullJobReconcileHandler)(nil)), "GetJob",
			func(_ctx context.Context, _req controllerruntime.Request) (*operationsv1alpha2.ImagePrePullJob, error) {
				return &operationsv1alpha2.ImagePrePullJob{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-job",
					},
					Status: operationsv1alpha2.ImagePrePullJobStatus{
						Phase: operationsv1alpha2.JobPhaseInProgress,
					},
				}, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*ImagePrePullJobReconcileHandler)(nil)), "UpdateJobStatus",
			func(_ctx context.Context, _job *operationsv1alpha2.ImagePrePullJob) error {
				updateJobStatusCalled = true
				return nil
			})

		handler := &ImagePrePullJobReconcileHandler{}
		err := handler.CheckTimeout(ctx, "test-job")
		require.NoError(t, err)
		assert.False(t, updateJobStatusCalled)
	})

	t.Run("the status of node tasks does not need timeout processing", func(t *testing.T) {
		var updateJobStatusCalled bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*ImagePrePullJobReconcileHandler)(nil)), "GetJob",
			func(_ctx context.Context, _req controllerruntime.Request) (*operationsv1alpha2.ImagePrePullJob, error) {
				ts := uint32(10)
				obj := &operationsv1alpha2.ImagePrePullJob{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-job",
					},
					Spec: operationsv1alpha2.ImagePrePullJobSpec{
						ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
							TimeoutSeconds: &ts,
						},
					},
					Status: operationsv1alpha2.ImagePrePullJobStatus{
						Phase: operationsv1alpha2.JobPhaseInProgress,
						NodeStatus: []operationsv1alpha2.ImagePrePullNodeTaskStatus{
							{Phase: operationsv1alpha2.NodeTaskPhaseSuccessful},
							{Phase: operationsv1alpha2.NodeTaskPhaseFailure},
							{Phase: operationsv1alpha2.NodeTaskPhaseUnknown},
						},
					},
				}
				return obj, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*ImagePrePullJobReconcileHandler)(nil)), "UpdateJobStatus",
			func(_ctx context.Context, _job *operationsv1alpha2.ImagePrePullJob) error {
				updateJobStatusCalled = true
				return nil
			})

		handler := &ImagePrePullJobReconcileHandler{}
		err := handler.CheckTimeout(ctx, "test-job")
		require.NoError(t, err)
		assert.False(t, updateJobStatusCalled)
	})

	t.Run("the node task has timed out due to last action time.", func(t *testing.T) {
		var updateJobStatusCalled bool
		ts := uint32(10)
		obj := &operationsv1alpha2.ImagePrePullJob{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-job",
			},
			Spec: operationsv1alpha2.ImagePrePullJobSpec{
				ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
					TimeoutSeconds: &ts,
				},
			},
			Status: operationsv1alpha2.ImagePrePullJobStatus{
				Phase: operationsv1alpha2.JobPhaseInProgress,
				NodeStatus: []operationsv1alpha2.ImagePrePullNodeTaskStatus{
					{
						Phase: operationsv1alpha2.NodeTaskPhaseInProgress,
						ActionFlow: []operationsv1alpha2.ImagePrePullJobActionStatus{
							{
								Action: operationsv1alpha2.ImagePrePullJobActionPull,
								Time:   time.Now().UTC().Add(-time.Second * 8).Format(time.RFC3339),
							},
						},
					},
				},
			},
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*ImagePrePullJobReconcileHandler)(nil)), "GetJob",
			func(_ctx context.Context, _req controllerruntime.Request) (*operationsv1alpha2.ImagePrePullJob, error) {
				return obj, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*ImagePrePullJobReconcileHandler)(nil)), "UpdateJobStatus",
			func(_ctx context.Context, _job *operationsv1alpha2.ImagePrePullJob) error {
				updateJobStatusCalled = true
				return nil
			})

		handler := &ImagePrePullJobReconcileHandler{}
		err := handler.CheckTimeout(ctx, "test-job")
		require.NoError(t, err)
		assert.False(t, updateJobStatusCalled)

		// Wait for time out
		time.Sleep(2 * time.Second)
		err = handler.CheckTimeout(ctx, "test-job")
		require.NoError(t, err)
		assert.True(t, updateJobStatusCalled)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhaseUnknown, obj.Status.NodeStatus[0].Phase)
		assert.Equal(t, NodeTaskReasonTimeout, obj.Status.NodeStatus[0].Reason)
	})

	t.Run("no any actions, the node task has timed out due to creation time.", func(t *testing.T) {
		var updateJobStatusCalled bool
		ts := uint32(10)
		obj := &operationsv1alpha2.ImagePrePullJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-job",
				CreationTimestamp: metav1.Time{Time: time.Now().UTC().Add(-time.Second * 8)},
			},
			Spec: operationsv1alpha2.ImagePrePullJobSpec{
				ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
					TimeoutSeconds: &ts,
				},
			},
			Status: operationsv1alpha2.ImagePrePullJobStatus{
				Phase: operationsv1alpha2.JobPhaseInProgress,
				NodeStatus: []operationsv1alpha2.ImagePrePullNodeTaskStatus{
					{
						Phase:      operationsv1alpha2.NodeTaskPhaseInProgress,
						ActionFlow: []operationsv1alpha2.ImagePrePullJobActionStatus{},
					},
				},
			},
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*ImagePrePullJobReconcileHandler)(nil)), "GetJob",
			func(_ctx context.Context, _req controllerruntime.Request) (*operationsv1alpha2.ImagePrePullJob, error) {
				return obj, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*ImagePrePullJobReconcileHandler)(nil)), "UpdateJobStatus",
			func(_ctx context.Context, _job *operationsv1alpha2.ImagePrePullJob) error {
				updateJobStatusCalled = true
				return nil
			})

		handler := &ImagePrePullJobReconcileHandler{}
		err := handler.CheckTimeout(ctx, "test-job")
		require.NoError(t, err)
		assert.False(t, updateJobStatusCalled)

		// Wait for time out
		time.Sleep(2 * time.Second)
		err = handler.CheckTimeout(ctx, "test-job")
		require.NoError(t, err)
		assert.True(t, updateJobStatusCalled)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhaseUnknown, obj.Status.NodeStatus[0].Phase)
		assert.Equal(t, NodeTaskReasonTimeout, obj.Status.NodeStatus[0].Reason)
	})
}

func fakeImagePrePullJobClient(objs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(operationsv1alpha2.SchemeGroupVersion,
		&operationsv1alpha2.ImagePrePullJob{},
		&operationsv1alpha2.ImagePrePullJobList{},
	)
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(objs...).
		Build()
}
