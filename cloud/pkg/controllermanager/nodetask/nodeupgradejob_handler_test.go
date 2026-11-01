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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

func TestNodeUpgradeJobGetJob(t *testing.T) {
	ctx := context.TODO()
	cli := fakeNodeUpgradeJobClient(&operationsv1alpha2.NodeUpgradeJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job"},
	})
	t.Run("not found", func(t *testing.T) {
		handler := NewNodeUpgradeJobReconcileHandler(cli, nil, nil)
		obj, err := handler.GetJob(ctx, controllerruntime.Request{
			NamespacedName: client.ObjectKey{Name: "not-found"},
		})
		assert.NoError(t, err)
		assert.Nil(t, obj)
	})

	t.Run("get job successful", func(t *testing.T) {
		handler := NewNodeUpgradeJobReconcileHandler(cli, nil, nil)
		obj, err := handler.GetJob(ctx, controllerruntime.Request{
			NamespacedName: client.ObjectKey{Name: "test-job"},
		})
		assert.NoError(t, err)
		assert.NotNil(t, obj)
	})
}

func TestNodeUpgradeJobFinalizer(t *testing.T) {
	ctx := context.TODO()
	cli := fakeNodeUpgradeJobClient(&operationsv1alpha2.NodeUpgradeJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-job",
		},
	})
	handler := NewNodeUpgradeJobReconcileHandler(cli, nil, nil)

	var found operationsv1alpha2.NodeUpgradeJob
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

func TestNodeUpgradeJobNotInitialized(t *testing.T) {
	cases := []struct {
		name string
		obj  *operationsv1alpha2.NodeUpgradeJob
		want bool
	}{
		{
			name: "phase is empty",
			obj:  &operationsv1alpha2.NodeUpgradeJob{},
			want: true,
		},
		{
			name: "phase not emnpty",
			obj: &operationsv1alpha2.NodeUpgradeJob{
				Status: operationsv1alpha2.NodeUpgradeJobStatus{
					Phase: operationsv1alpha2.JobPhaseInit,
				},
			},
			want: false,
		},
	}

	handler := NewNodeUpgradeJobReconcileHandler(nil, nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, handler.NotInitialized(c.obj))
		})
	}
}

func TestNodeUpgradeJobInitNodesStatus(t *testing.T) {
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

		handler := NewNodeUpgradeJobReconcileHandler(nil, nil, nil)
		job := &operationsv1alpha2.NodeUpgradeJob{}
		handler.InitNodesStatus(ctx, job)
		assert.Equal(t, operationsv1alpha2.JobPhaseFailure, job.Status.Phase)
		assert.Equal(t, "failed to verify node define", job.Status.Reason)
		cond := meta.FindStatusCondition(job.Status.Conditions, operationsv1alpha2.NodeUpgradeJobConditionFailed)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, nodeUpgradeJobReasonNodeVerifyFailed, cond.Reason)
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

		handler := NewNodeUpgradeJobReconcileHandler(nil, nil, nil)
		job := &operationsv1alpha2.NodeUpgradeJob{}
		handler.InitNodesStatus(ctx, job)
		assert.Equal(t, operationsv1alpha2.JobPhaseInit, job.Status.Phase)
		assert.Len(t, job.Status.NodeStatus, 2)
		cond := meta.FindStatusCondition(job.Status.Conditions, operationsv1alpha2.NodeUpgradeJobConditionInitialized)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)

		assert.Equal(t, "node1", job.Status.NodeStatus[0].NodeName)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhasePending, job.Status.NodeStatus[0].Phase)
		assert.Empty(t, job.Status.NodeStatus[0].Reason)

		assert.Equal(t, "node2", job.Status.NodeStatus[1].NodeName)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhaseFailure, job.Status.NodeStatus[1].Phase)
		assert.Equal(t, "failed to init node2", job.Status.NodeStatus[1].Reason)
	})
}

func TestNodeUpgradeJobIsFinalPhase(t *testing.T) {
	cases := []struct {
		name string
		obj  *operationsv1alpha2.NodeUpgradeJob
		want bool
	}{
		{
			name: "init is not a final phase",
			obj: &operationsv1alpha2.NodeUpgradeJob{
				Status: operationsv1alpha2.NodeUpgradeJobStatus{
					Phase: operationsv1alpha2.JobPhaseInProgress,
				},
			},
			want: false,
		},
		{
			name: "failure is not a final phase",
			obj: &operationsv1alpha2.NodeUpgradeJob{
				Status: operationsv1alpha2.NodeUpgradeJobStatus{
					Phase: operationsv1alpha2.JobPhaseFailure,
				},
			},
			want: false,
		},
		{
			name: "completed is a final phase",
			obj: &operationsv1alpha2.NodeUpgradeJob{
				Status: operationsv1alpha2.NodeUpgradeJobStatus{
					Phase: operationsv1alpha2.JobPhaseCompleted,
				},
			},
			want: true,
		},
	}

	handler := NewNodeUpgradeJobReconcileHandler(nil, nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, handler.IsFinalPhase(c.obj))
		})
	}
}

func TestNodeUpgradeJobIsDeleted(t *testing.T) {
	cases := []struct {
		name string
		obj  *operationsv1alpha2.NodeUpgradeJob
		want bool
	}{
		{
			name: "not deleted",
			obj: &operationsv1alpha2.NodeUpgradeJob{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-job",
				},
			},
			want: false,
		},
		{
			name: "deleted",
			obj: &operationsv1alpha2.NodeUpgradeJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-job",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
			},
			want: true,
		},
	}
	handler := NewNodeUpgradeJobReconcileHandler(nil, nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, handler.IsDeleted(c.obj))
		})
	}
}

func TestNodeUpgradeJobCalculateStatus(t *testing.T) {
	cases := []struct {
		name        string
		obj         *operationsv1alpha2.NodeUpgradeJob
		wantChanged bool
		wantPhase   operationsv1alpha2.JobPhase
		wantCond    string
	}{
		{
			name: "some node tasks are in progress",
			obj: &operationsv1alpha2.NodeUpgradeJob{
				Status: operationsv1alpha2.NodeUpgradeJobStatus{
					Phase: operationsv1alpha2.JobPhaseInProgress,
					NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
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
			wantChanged: true,
			wantPhase:   operationsv1alpha2.JobPhaseInProgress,
			wantCond:    operationsv1alpha2.NodeUpgradeJobConditionInProgress,
		},
		{
			name: "most node task are failure",
			obj: &operationsv1alpha2.NodeUpgradeJob{
				Spec: operationsv1alpha2.NodeUpgradeJobSpec{
					FailureTolerate: "0.5",
				},
				Status: operationsv1alpha2.NodeUpgradeJobStatus{
					Phase: operationsv1alpha2.JobPhaseInProgress,
					NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
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
			wantCond:    operationsv1alpha2.NodeUpgradeJobConditionFailed,
		},
		{
			name: "most node task are successful",
			obj: &operationsv1alpha2.NodeUpgradeJob{
				Spec: operationsv1alpha2.NodeUpgradeJobSpec{
					FailureTolerate: "0.5",
				},
				Status: operationsv1alpha2.NodeUpgradeJobStatus{
					Phase: operationsv1alpha2.JobPhaseInProgress,
					NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
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
			wantCond:    operationsv1alpha2.NodeUpgradeJobConditionPartiallySucceeded,
		},
	}

	ctx := context.TODO()
	handler := NewNodeUpgradeJobReconcileHandler(nil, nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			changed := handler.CalculateStatus(ctx, c.obj)
			assert.Equal(t, c.wantChanged, changed)
			assert.Equal(t, c.wantPhase, c.obj.Status.Phase)
			cond := meta.FindStatusCondition(c.obj.Status.Conditions, c.wantCond)
			require.NotNil(t, cond)
			assert.Equal(t, metav1.ConditionTrue, cond.Status)
		})
	}
}

func TestNodeUpgradeJobUpdateJobStatus(t *testing.T) {
	ctx := context.TODO()
	job := &operationsv1alpha2.NodeUpgradeJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job"},
	}
	cli := fakeNodeUpgradeJobClient(job)
	handler := NewNodeUpgradeJobReconcileHandler(cli, nil, nil)
	job.Status.Phase = operationsv1alpha2.JobPhaseInProgress
	err := handler.UpdateJobStatus(ctx, job)
	assert.NoError(t, err)
}

func TestNodeUpgradeJobCheckTimeout(t *testing.T) {
	ctx := context.TODO()

	t.Run("job is not in progress", func(t *testing.T) {
		var updateJobStatusCalled bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "GetJob",
			func(_ctx context.Context, _req controllerruntime.Request) (*operationsv1alpha2.NodeUpgradeJob, error) {
				return &operationsv1alpha2.NodeUpgradeJob{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-job",
					},
					Status: operationsv1alpha2.NodeUpgradeJobStatus{
						Phase: operationsv1alpha2.JobPhaseInit,
					},
				}, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "UpdateJobStatus",
			func(_ctx context.Context, _job *operationsv1alpha2.NodeUpgradeJob) error {
				updateJobStatusCalled = true
				return nil
			})

		handler := &NodeUpgradeJobReconcileHandler{}
		err := handler.CheckTimeout(ctx, "test-job")
		require.NoError(t, err)
		assert.False(t, updateJobStatusCalled)
	})

	t.Run("timeout seconds is not set", func(t *testing.T) {
		var updateJobStatusCalled bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "GetJob",
			func(_ctx context.Context, _req controllerruntime.Request) (*operationsv1alpha2.NodeUpgradeJob, error) {
				return &operationsv1alpha2.NodeUpgradeJob{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-job",
					},
					Status: operationsv1alpha2.NodeUpgradeJobStatus{
						Phase: operationsv1alpha2.JobPhaseInProgress,
					},
				}, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "UpdateJobStatus",
			func(_ctx context.Context, _job *operationsv1alpha2.NodeUpgradeJob) error {
				updateJobStatusCalled = true
				return nil
			})

		handler := &NodeUpgradeJobReconcileHandler{}
		err := handler.CheckTimeout(ctx, "test-job")
		require.NoError(t, err)
		assert.False(t, updateJobStatusCalled)
	})

	t.Run("the status of node tasks does not need timeout processing", func(t *testing.T) {
		var updateJobStatusCalled bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "GetJob",
			func(_ctx context.Context, _req controllerruntime.Request) (*operationsv1alpha2.NodeUpgradeJob, error) {
				ts := uint32(10)
				obj := &operationsv1alpha2.NodeUpgradeJob{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-job",
					},
					Spec: operationsv1alpha2.NodeUpgradeJobSpec{
						TimeoutSeconds: &ts,
					},
					Status: operationsv1alpha2.NodeUpgradeJobStatus{
						Phase: operationsv1alpha2.JobPhaseInProgress,
						NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
							{Phase: operationsv1alpha2.NodeTaskPhaseSuccessful},
							{Phase: operationsv1alpha2.NodeTaskPhaseFailure},
							{Phase: operationsv1alpha2.NodeTaskPhaseUnknown},
						},
					},
				}
				return obj, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "UpdateJobStatus",
			func(_ctx context.Context, _job *operationsv1alpha2.NodeUpgradeJob) error {
				updateJobStatusCalled = true
				return nil
			})

		handler := &NodeUpgradeJobReconcileHandler{}
		err := handler.CheckTimeout(ctx, "test-job")
		require.NoError(t, err)
		assert.False(t, updateJobStatusCalled)
	})

	t.Run("the node task has timed out due to last action time.", func(t *testing.T) {
		var updateJobStatusCalled bool
		ts := uint32(10)
		obj := &operationsv1alpha2.NodeUpgradeJob{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-job",
			},
			Spec: operationsv1alpha2.NodeUpgradeJobSpec{
				TimeoutSeconds: &ts,
			},
			Status: operationsv1alpha2.NodeUpgradeJobStatus{
				Phase: operationsv1alpha2.JobPhaseInProgress,
				NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
					{
						Phase: operationsv1alpha2.NodeTaskPhaseInProgress,
						ActionFlow: []operationsv1alpha2.NodeUpgradeJobActionStatus{
							{
								Action: operationsv1alpha2.NodeUpgradeJobActionUpgrade,
								Time:   time.Now().UTC().Add(-time.Second * 8).Format(time.RFC3339),
							},
						},
					},
				},
			},
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "GetJob",
			func(_ctx context.Context, _req controllerruntime.Request) (*operationsv1alpha2.NodeUpgradeJob, error) {
				return obj, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "UpdateJobStatus",
			func(_ctx context.Context, _job *operationsv1alpha2.NodeUpgradeJob) error {
				updateJobStatusCalled = true
				return nil
			})

		handler := &NodeUpgradeJobReconcileHandler{}
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
		cond := meta.FindStatusCondition(obj.Status.Conditions, operationsv1alpha2.NodeUpgradeJobConditionTimedOut)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
	})

	t.Run("no any actions, the node task has timed out due to creation time.", func(t *testing.T) {
		var updateJobStatusCalled bool
		ts := uint32(10)
		obj := &operationsv1alpha2.NodeUpgradeJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-job",
				CreationTimestamp: metav1.Time{Time: time.Now().UTC().Add(-time.Second * 8)},
			},
			Spec: operationsv1alpha2.NodeUpgradeJobSpec{
				TimeoutSeconds: &ts,
			},
			Status: operationsv1alpha2.NodeUpgradeJobStatus{
				Phase: operationsv1alpha2.JobPhaseInProgress,
				NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
					{
						Phase:      operationsv1alpha2.NodeTaskPhaseInProgress,
						ActionFlow: []operationsv1alpha2.NodeUpgradeJobActionStatus{},
					},
				},
			},
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "GetJob",
			func(_ctx context.Context, _req controllerruntime.Request) (*operationsv1alpha2.NodeUpgradeJob, error) {
				return obj, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "UpdateJobStatus",
			func(_ctx context.Context, _job *operationsv1alpha2.NodeUpgradeJob) error {
				updateJobStatusCalled = true
				return nil
			})

		handler := &NodeUpgradeJobReconcileHandler{}
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
		cond := meta.FindStatusCondition(obj.Status.Conditions, operationsv1alpha2.NodeUpgradeJobConditionTimedOut)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
	})
}

func fakeNodeUpgradeJobClient(objs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(operationsv1alpha2.SchemeGroupVersion,
		&operationsv1alpha2.NodeUpgradeJob{},
		&operationsv1alpha2.NodeUpgradeJobList{},
	)
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(objs...).
		Build()
}

func TestNodeUpgradeJobEvents(t *testing.T) {
	ctx := context.TODO()

	t.Run("Event verification for InitNodesStatus", func(t *testing.T) {
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

		recorder := record.NewFakeRecorder(100)
		handler := NewNodeUpgradeJobReconcileHandler(nil, nil, recorder)
		job := &operationsv1alpha2.NodeUpgradeJob{}
		handler.InitNodesStatus(ctx, job)

		select {
		case ev := <-recorder.Events:
			assert.Contains(t, ev, "Warning NodeVerificationFailed failed to verify node define")
		default:
			t.Fatal("expected Warning NodeVerificationFailed event")
		}

		patches.Reset()
		patches.ApplyFunc(VerifyNodeDefine,
			func(ctx context.Context,
				che cache.Cache,
				nodeNames []string,
				nodeSelector *metav1.LabelSelector,
			) (res []NodeVerificationResult, err error) {
				return []NodeVerificationResult{{NodeName: "node1"}}, nil
			})

		recorder = record.NewFakeRecorder(100)
		handler = NewNodeUpgradeJobReconcileHandler(nil, nil, recorder)
		job = &operationsv1alpha2.NodeUpgradeJob{}
		handler.InitNodesStatus(ctx, job)

		select {
		case ev := <-recorder.Events:
			assert.Contains(t, ev, "Normal Initialized Node upgrade job selected target nodes and initialized node task status.")
		default:
			t.Fatal("expected Normal Initialized event")
		}
	})

	t.Run("Event verification for CalculateStatus transitions", func(t *testing.T) {
		recorder := record.NewFakeRecorder(100)
		handler := NewNodeUpgradeJobReconcileHandler(nil, nil, recorder)

		// 1. InProgress transition
		job := &operationsv1alpha2.NodeUpgradeJob{
			Status: operationsv1alpha2.NodeUpgradeJobStatus{
				Phase: operationsv1alpha2.JobPhaseInit,
				NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
					{NodeName: "node1", Phase: operationsv1alpha2.NodeTaskPhaseInProgress},
				},
			},
		}
		changed := handler.CalculateStatus(ctx, job)
		assert.True(t, changed)
		select {
		case ev := <-recorder.Events:
			assert.Contains(t, ev, "Normal InProgress Node upgrade job is now in progress.")
		default:
			t.Fatal("expected Normal InProgress event")
		}

		// 2. Completed transition (Success)
		job = &operationsv1alpha2.NodeUpgradeJob{
			Status: operationsv1alpha2.NodeUpgradeJobStatus{
				Phase: operationsv1alpha2.JobPhaseInProgress,
				NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
					{NodeName: "node1", Phase: operationsv1alpha2.NodeTaskPhaseSuccessful},
				},
			},
		}
		changed = handler.CalculateStatus(ctx, job)
		assert.True(t, changed)
		select {
		case ev := <-recorder.Events:
			assert.Contains(t, ev, "Normal Completed Node upgrade job has successfully completed.")
		default:
			t.Fatal("expected Normal Completed event")
		}

		// 3. PartiallySucceeded transition
		job = &operationsv1alpha2.NodeUpgradeJob{
			Spec: operationsv1alpha2.NodeUpgradeJobSpec{
				FailureTolerate: "0.5",
			},
			Status: operationsv1alpha2.NodeUpgradeJobStatus{
				Phase: operationsv1alpha2.JobPhaseInProgress,
				NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
					{NodeName: "node1", Phase: operationsv1alpha2.NodeTaskPhaseSuccessful},
					{NodeName: "node2", Phase: operationsv1alpha2.NodeTaskPhaseFailure},
				},
			},
		}
		changed = handler.CalculateStatus(ctx, job)
		assert.True(t, changed)
		select {
		case ev := <-recorder.Events:
			assert.Contains(t, ev, "Normal PartiallySucceeded Node upgrade job completed with failed nodes within the failure tolerance.")
		default:
			t.Fatal("expected Normal PartiallySucceeded event")
		}

		// 4. Failed transition
		job = &operationsv1alpha2.NodeUpgradeJob{
			Spec: operationsv1alpha2.NodeUpgradeJobSpec{
				FailureTolerate: "0.1",
			},
			Status: operationsv1alpha2.NodeUpgradeJobStatus{
				Phase: operationsv1alpha2.JobPhaseInProgress,
				NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
					{NodeName: "node1", Phase: operationsv1alpha2.NodeTaskPhaseFailure},
				},
			},
		}
		changed = handler.CalculateStatus(ctx, job)
		assert.True(t, changed)
		select {
		case ev := <-recorder.Events:
			assert.Contains(t, ev, "Warning Failed the number of failed nodes is 1/1, which exceeds the failure tolerance threshold")
		default:
			t.Fatal("expected Warning Failed event")
		}
	})

	t.Run("Event verification for CheckTimeout transitions and deduplication", func(t *testing.T) {
		jobName := "test-job"
		ts := uint32(1)
		obj := &operationsv1alpha2.NodeUpgradeJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              jobName,
				CreationTimestamp: metav1.Time{Time: time.Now().UTC().Add(-10 * time.Second)},
			},
			Spec: operationsv1alpha2.NodeUpgradeJobSpec{
				TimeoutSeconds: &ts,
			},
			Status: operationsv1alpha2.NodeUpgradeJobStatus{
				Phase: operationsv1alpha2.JobPhaseInProgress,
				NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
					{
						NodeName: "node1",
						Phase:    operationsv1alpha2.NodeTaskPhaseInProgress,
					},
				},
			},
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "GetJob",
			func(_ctx context.Context, _req controllerruntime.Request) (*operationsv1alpha2.NodeUpgradeJob, error) {
				return obj, nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf((*NodeUpgradeJobReconcileHandler)(nil)), "UpdateJobStatus",
			func(_ctx context.Context, _job *operationsv1alpha2.NodeUpgradeJob) error {
				return nil
			})

		recorder := record.NewFakeRecorder(100)
		handler := NewNodeUpgradeJobReconcileHandler(nil, nil, recorder)

		err := handler.CheckTimeout(ctx, jobName)
		require.NoError(t, err)

		select {
		case ev := <-recorder.Events:
			assert.Contains(t, ev, "Warning TimedOut Node upgrade job has timed out.")
		default:
			t.Fatal("expected Warning TimedOut event")
		}

		// Re-run should NOT emit duplicate event
		err = handler.CheckTimeout(ctx, jobName)
		require.NoError(t, err)

		select {
		case ev := <-recorder.Events:
			t.Fatalf("unexpected duplicate event: %s", ev)
		default:
			// Success, no duplicate event!
		}
	})
}
