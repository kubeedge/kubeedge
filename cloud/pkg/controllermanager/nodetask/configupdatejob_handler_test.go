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

package nodetask

import (
	"context"
	"errors"
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
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

func TestConfigUpdateJobGetResource(t *testing.T) {
	handler := NewConfigUpdateJobReconcileHandler(nil, nil)
	assert.Equal(t, operationsv1alpha2.ResourceConfigUpdateJob, handler.GetResource())
}

func TestConfigUpdateJobGetJob(t *testing.T) {
	ctx := context.TODO()
	cli := fakeConfigUpdateJobClient(&operationsv1alpha2.ConfigUpdateJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job"},
	})

	t.Run("not found", func(t *testing.T) {
		handler := NewConfigUpdateJobReconcileHandler(cli, nil)
		obj, err := handler.GetJob(ctx, controllerruntime.Request{
			NamespacedName: client.ObjectKey{Name: "not-found"},
		})
		assert.NoError(t, err)
		assert.Nil(t, obj)
	})

	t.Run("get job successful", func(t *testing.T) {
		handler := NewConfigUpdateJobReconcileHandler(cli, nil)
		obj, err := handler.GetJob(ctx, controllerruntime.Request{
			NamespacedName: client.ObjectKey{Name: "test-job"},
		})
		require.NoError(t, err)
		require.NotNil(t, obj)
		assert.Equal(t, "test-job", obj.Name)
	})

	t.Run("get job failed", func(t *testing.T) {
		// The ConfigUpdateJob type is not registered in the scheme, so the client
		// returns an error that is not a NotFound error.
		cli := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		handler := NewConfigUpdateJobReconcileHandler(cli, nil)
		obj, err := handler.GetJob(ctx, controllerruntime.Request{
			NamespacedName: client.ObjectKey{Name: "test-job"},
		})
		assert.ErrorContains(t, err, "failed to get")
		assert.Nil(t, obj)
	})
}

func TestConfigUpdateJobFinalizer(t *testing.T) {
	ctx := context.TODO()
	cli := fakeConfigUpdateJobClient(&operationsv1alpha2.ConfigUpdateJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job"},
	})
	handler := NewConfigUpdateJobReconcileHandler(cli, nil)

	var found operationsv1alpha2.ConfigUpdateJob
	err := cli.Get(ctx, client.ObjectKey{Name: "test-job"}, &found)
	require.NoError(t, err)
	assert.True(t, handler.NoFinalizer(&found))

	err = handler.AddFinalizer(ctx, &found)
	require.NoError(t, err)
	err = cli.Get(ctx, client.ObjectKey{Name: "test-job"}, &found)
	require.NoError(t, err)
	assert.False(t, handler.NoFinalizer(&found))
	assert.Contains(t, found.Finalizers, operationsv1alpha2.FinalizerConfigUpdateJob)

	err = handler.RemoveFinalizer(ctx, &found)
	require.NoError(t, err)
	err = cli.Get(ctx, client.ObjectKey{Name: "test-job"}, &found)
	require.NoError(t, err)
	assert.True(t, handler.NoFinalizer(&found))
}

func TestConfigUpdateJobNotInitialized(t *testing.T) {
	cases := []struct {
		name string
		obj  *operationsv1alpha2.ConfigUpdateJob
		want bool
	}{
		{
			name: "phase is empty",
			obj:  &operationsv1alpha2.ConfigUpdateJob{},
			want: true,
		},
		{
			name: "phase is not empty",
			obj: &operationsv1alpha2.ConfigUpdateJob{
				Status: operationsv1alpha2.ConfigUpdateJobStatus{
					Phase: operationsv1alpha2.JobPhaseInit,
				},
			},
			want: false,
		},
	}

	handler := NewConfigUpdateJobReconcileHandler(nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, handler.NotInitialized(c.obj))
		})
	}
}

func TestConfigUpdateJobInitNodesStatus(t *testing.T) {
	ctx := context.TODO()

	t.Run("verify node define failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(VerifyNodeDefine,
			func(_ctx context.Context,
				_che cache.Cache,
				_nodeNames []string,
				_nodeSelector *metav1.LabelSelector,
			) ([]NodeVerificationResult, error) {
				return nil, errors.New("failed to verify node define")
			})

		handler := NewConfigUpdateJobReconcileHandler(nil, nil)
		job := &operationsv1alpha2.ConfigUpdateJob{}
		handler.InitNodesStatus(ctx, job)
		assert.Equal(t, operationsv1alpha2.JobPhaseFailure, job.Status.Phase)
		assert.Equal(t, "failed to verify node define", job.Status.Reason)
		assert.Empty(t, job.Status.NodeStatus)
	})

	t.Run("init nodes status successful", func(t *testing.T) {
		var (
			gotNodeNames []string
			gotSelector  *metav1.LabelSelector
		)
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(VerifyNodeDefine,
			func(_ctx context.Context,
				_che cache.Cache,
				nodeNames []string,
				nodeSelector *metav1.LabelSelector,
			) ([]NodeVerificationResult, error) {
				gotNodeNames, gotSelector = nodeNames, nodeSelector
				return []NodeVerificationResult{
					{NodeName: "node1"},
					{NodeName: "node2", ErrorMessage: "failed to init node2"},
				}, nil
			})

		selector := &metav1.LabelSelector{MatchLabels: map[string]string{"app": "edge"}}
		handler := NewConfigUpdateJobReconcileHandler(nil, nil)
		job := &operationsv1alpha2.ConfigUpdateJob{
			Spec: operationsv1alpha2.ConfigUpdateJobSpec{
				NodeNames:     []string{"node1", "node2"},
				LabelSelector: selector,
			},
		}
		handler.InitNodesStatus(ctx, job)

		// The node names and label selector of the job are used to find the nodes.
		assert.Equal(t, []string{"node1", "node2"}, gotNodeNames)
		assert.Equal(t, selector, gotSelector)

		assert.Equal(t, operationsv1alpha2.JobPhaseInit, job.Status.Phase)
		require.Len(t, job.Status.NodeStatus, 2)

		assert.Equal(t, "node1", job.Status.NodeStatus[0].NodeName)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhasePending, job.Status.NodeStatus[0].Phase)
		assert.Empty(t, job.Status.NodeStatus[0].Reason)

		assert.Equal(t, "node2", job.Status.NodeStatus[1].NodeName)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhaseFailure, job.Status.NodeStatus[1].Phase)
		assert.Equal(t, "failed to init node2", job.Status.NodeStatus[1].Reason)
	})
}

func TestConfigUpdateJobIsFinalPhase(t *testing.T) {
	cases := []struct {
		name  string
		phase operationsv1alpha2.JobPhase
		want  bool
	}{
		{name: "not initialized is not a final phase", phase: "", want: false},
		{name: "init is not a final phase", phase: operationsv1alpha2.JobPhaseInit, want: false},
		{name: "in progress is not a final phase", phase: operationsv1alpha2.JobPhaseInProgress, want: false},
		{name: "completed is a final phase", phase: operationsv1alpha2.JobPhaseCompleted, want: true},
		{name: "failure is a final phase", phase: operationsv1alpha2.JobPhaseFailure, want: true},
	}

	handler := NewConfigUpdateJobReconcileHandler(nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			job := &operationsv1alpha2.ConfigUpdateJob{
				Status: operationsv1alpha2.ConfigUpdateJobStatus{Phase: c.phase},
			}
			assert.Equal(t, c.want, handler.IsFinalPhase(job))
		})
	}
}

func TestConfigUpdateJobIsDeleted(t *testing.T) {
	cases := []struct {
		name string
		obj  *operationsv1alpha2.ConfigUpdateJob
		want bool
	}{
		{
			name: "not deleted",
			obj: &operationsv1alpha2.ConfigUpdateJob{
				ObjectMeta: metav1.ObjectMeta{Name: "test-job"},
			},
			want: false,
		},
		{
			name: "deleted",
			obj: &operationsv1alpha2.ConfigUpdateJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-job",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
			},
			want: true,
		},
	}

	handler := NewConfigUpdateJobReconcileHandler(nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, handler.IsDeleted(c.obj))
		})
	}
}

func TestConfigUpdateJobCalculateStatus(t *testing.T) {
	cases := []struct {
		name            string
		failureTolerate string
		phase           operationsv1alpha2.JobPhase
		reason          string
		nodePhases      []operationsv1alpha2.NodeTaskPhase
		wantChanged     bool
		wantPhase       operationsv1alpha2.JobPhase
		wantReason      string
	}{
		{
			name:  "some node tasks are in progress",
			phase: operationsv1alpha2.JobPhaseInProgress,
			nodePhases: []operationsv1alpha2.NodeTaskPhase{
				operationsv1alpha2.NodeTaskPhasePending,
				operationsv1alpha2.NodeTaskPhaseInProgress,
				operationsv1alpha2.NodeTaskPhaseSuccessful,
				operationsv1alpha2.NodeTaskPhaseFailure,
			},
			wantChanged: false,
			wantPhase:   operationsv1alpha2.JobPhaseInProgress,
		},
		{
			name:            "failed nodes exceed the failure tolerance",
			failureTolerate: "0.5",
			phase:           operationsv1alpha2.JobPhaseInProgress,
			nodePhases: []operationsv1alpha2.NodeTaskPhase{
				operationsv1alpha2.NodeTaskPhaseUnknown,
				operationsv1alpha2.NodeTaskPhaseSuccessful,
				operationsv1alpha2.NodeTaskPhaseFailure,
			},
			wantChanged: true,
			wantPhase:   operationsv1alpha2.JobPhaseFailure,
			wantReason:  "the number of failed nodes is 2/3, which exceeds the failure tolerance threshold",
		},
		{
			name:            "failed nodes are within the failure tolerance",
			failureTolerate: "0.5",
			phase:           operationsv1alpha2.JobPhaseInProgress,
			nodePhases: []operationsv1alpha2.NodeTaskPhase{
				operationsv1alpha2.NodeTaskPhaseSuccessful,
				operationsv1alpha2.NodeTaskPhaseSuccessful,
				operationsv1alpha2.NodeTaskPhaseFailure,
			},
			wantChanged: true,
			wantPhase:   operationsv1alpha2.JobPhaseCompleted,
		},
		{
			name:  "any failed node fails the job without failure tolerance",
			phase: operationsv1alpha2.JobPhaseInProgress,
			nodePhases: []operationsv1alpha2.NodeTaskPhase{
				operationsv1alpha2.NodeTaskPhaseSuccessful,
				operationsv1alpha2.NodeTaskPhaseFailure,
			},
			wantChanged: true,
			wantPhase:   operationsv1alpha2.JobPhaseFailure,
			wantReason:  "the number of failed nodes is 1/2, which exceeds the failure tolerance threshold",
		},
		{
			name:   "stale reason is cleared when the phase is unchanged",
			phase:  operationsv1alpha2.JobPhaseInProgress,
			reason: "stale reason",
			nodePhases: []operationsv1alpha2.NodeTaskPhase{
				operationsv1alpha2.NodeTaskPhaseInProgress,
			},
			wantChanged: true,
			wantPhase:   operationsv1alpha2.JobPhaseInProgress,
		},
	}

	ctx := context.TODO()
	handler := NewConfigUpdateJobReconcileHandler(nil, nil)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			job := &operationsv1alpha2.ConfigUpdateJob{
				Spec: operationsv1alpha2.ConfigUpdateJobSpec{
					FailureTolerate: c.failureTolerate,
				},
				Status: operationsv1alpha2.ConfigUpdateJobStatus{
					Phase:  c.phase,
					Reason: c.reason,
				},
			}
			for _, phase := range c.nodePhases {
				job.Status.NodeStatus = append(job.Status.NodeStatus,
					operationsv1alpha2.ConfigUpdateJobNodeTaskStatus{Phase: phase})
			}

			changed := handler.CalculateStatus(ctx, job)
			assert.Equal(t, c.wantChanged, changed)
			assert.Equal(t, c.wantPhase, job.Status.Phase)
			assert.Equal(t, c.wantReason, job.Status.Reason)
		})
	}
}

func TestConfigUpdateJobUpdateJobStatus(t *testing.T) {
	ctx := context.TODO()

	t.Run("update job status successful", func(t *testing.T) {
		job := &operationsv1alpha2.ConfigUpdateJob{
			ObjectMeta: metav1.ObjectMeta{Name: "test-job"},
		}
		cli := fakeConfigUpdateJobClient(job)
		handler := NewConfigUpdateJobReconcileHandler(cli, nil)

		job.Status.Phase = operationsv1alpha2.JobPhaseInProgress
		require.NoError(t, handler.UpdateJobStatus(ctx, job))

		var found operationsv1alpha2.ConfigUpdateJob
		require.NoError(t, cli.Get(ctx, client.ObjectKey{Name: "test-job"}, &found))
		assert.Equal(t, operationsv1alpha2.JobPhaseInProgress, found.Status.Phase)
	})

	t.Run("update job status failed", func(t *testing.T) {
		cli := fakeConfigUpdateJobClient()
		handler := NewConfigUpdateJobReconcileHandler(cli, nil)
		job := &operationsv1alpha2.ConfigUpdateJob{
			ObjectMeta: metav1.ObjectMeta{Name: "not-found"},
		}
		err := handler.UpdateJobStatus(ctx, job)
		assert.ErrorContains(t, err, "failed to update configupdate job not-found status")
	})
}

func TestConfigUpdateJobCheckTimeout(t *testing.T) {
	const jobName = "test-job"
	var (
		ctx     = context.TODO()
		timeout = uint32(60)
		// Far enough from now that the result does not depend on how fast the test runs.
		expired = time.Now().UTC().Add(-10 * time.Minute)
		recent  = time.Now().UTC()
	)

	newJob := func(phase operationsv1alpha2.JobPhase, timeoutSeconds *uint32, createdAt time.Time,
		nodeStatus ...operationsv1alpha2.ConfigUpdateJobNodeTaskStatus,
	) *operationsv1alpha2.ConfigUpdateJob {
		return &operationsv1alpha2.ConfigUpdateJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              jobName,
				CreationTimestamp: metav1.Time{Time: createdAt},
			},
			Spec: operationsv1alpha2.ConfigUpdateJobSpec{
				TimeoutSeconds: timeoutSeconds,
			},
			Status: operationsv1alpha2.ConfigUpdateJobStatus{
				Phase:      phase,
				NodeStatus: nodeStatus,
			},
		}
	}
	nodeTask := func(phase operationsv1alpha2.NodeTaskPhase, actionTimes ...string,
	) operationsv1alpha2.ConfigUpdateJobNodeTaskStatus {
		status := operationsv1alpha2.ConfigUpdateJobNodeTaskStatus{
			NodeName: "node1",
			Phase:    phase,
		}
		for _, actionTime := range actionTimes {
			status.ActionFlow = append(status.ActionFlow, operationsv1alpha2.ConfigUpdateJobActionStatus{
				Action: operationsv1alpha2.ConfigUpdateJobActionUpdate,
				Time:   actionTime,
			})
		}
		return status
	}
	// getNodeStatus returns the node task status stored by the client after CheckTimeout.
	getNodeStatus := func(t *testing.T, cli client.Client) []operationsv1alpha2.ConfigUpdateJobNodeTaskStatus {
		var found operationsv1alpha2.ConfigUpdateJob
		require.NoError(t, cli.Get(ctx, client.ObjectKey{Name: jobName}, &found))
		return found.Status.NodeStatus
	}

	t.Run("failed to get job", func(t *testing.T) {
		cli := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		handler := NewConfigUpdateJobReconcileHandler(cli, nil)
		err := handler.CheckTimeout(ctx, jobName)
		assert.ErrorContains(t, err, "failed to get")
	})

	noTimeoutCases := []struct {
		name string
		job  *operationsv1alpha2.ConfigUpdateJob
	}{
		{
			name: "job is not in progress",
			job: newJob(operationsv1alpha2.JobPhaseInit, &timeout, expired,
				nodeTask(operationsv1alpha2.NodeTaskPhasePending)),
		},
		{
			name: "timeout seconds is not set",
			job: newJob(operationsv1alpha2.JobPhaseInProgress, nil, expired,
				nodeTask(operationsv1alpha2.NodeTaskPhasePending)),
		},
		{
			name: "timeout seconds is zero",
			job: newJob(operationsv1alpha2.JobPhaseInProgress, new(uint32), expired,
				nodeTask(operationsv1alpha2.NodeTaskPhasePending)),
		},
		{
			name: "node tasks are already finished",
			job: newJob(operationsv1alpha2.JobPhaseInProgress, &timeout, expired,
				nodeTask(operationsv1alpha2.NodeTaskPhaseSuccessful, expired.Format(time.RFC3339)),
				nodeTask(operationsv1alpha2.NodeTaskPhaseFailure, expired.Format(time.RFC3339)),
				nodeTask(operationsv1alpha2.NodeTaskPhaseUnknown, expired.Format(time.RFC3339))),
		},
		{
			name: "last action is within the timeout",
			job: newJob(operationsv1alpha2.JobPhaseInProgress, &timeout, expired,
				nodeTask(operationsv1alpha2.NodeTaskPhaseInProgress,
					expired.Format(time.RFC3339), recent.Format(time.RFC3339))),
		},
		{
			name: "no actions and the job is created within the timeout",
			job: newJob(operationsv1alpha2.JobPhaseInProgress, &timeout, recent,
				nodeTask(operationsv1alpha2.NodeTaskPhasePending)),
		},
	}
	for _, c := range noTimeoutCases {
		t.Run(c.name, func(t *testing.T) {
			want := c.job.DeepCopy().Status.NodeStatus
			cli := fakeConfigUpdateJobClient(c.job)
			handler := NewConfigUpdateJobReconcileHandler(cli, nil)

			require.NoError(t, handler.CheckTimeout(ctx, jobName))
			assert.Equal(t, want, getNodeStatus(t, cli))
		})
	}

	timeoutCases := []struct {
		name string
		job  *operationsv1alpha2.ConfigUpdateJob
	}{
		{
			name: "last action has timed out",
			job: newJob(operationsv1alpha2.JobPhaseInProgress, &timeout, recent,
				nodeTask(operationsv1alpha2.NodeTaskPhaseInProgress,
					recent.Format(time.RFC3339), expired.Format(time.RFC3339))),
		},
		{
			name: "no actions and the job is created before the timeout",
			job: newJob(operationsv1alpha2.JobPhaseInProgress, &timeout, expired,
				nodeTask(operationsv1alpha2.NodeTaskPhasePending)),
		},
	}
	for _, c := range timeoutCases {
		t.Run(c.name, func(t *testing.T) {
			cli := fakeConfigUpdateJobClient(c.job)
			handler := NewConfigUpdateJobReconcileHandler(cli, nil)

			require.NoError(t, handler.CheckTimeout(ctx, jobName))
			nodeStatus := getNodeStatus(t, cli)
			require.Len(t, nodeStatus, 1)
			assert.Equal(t, operationsv1alpha2.NodeTaskPhaseUnknown, nodeStatus[0].Phase)
			assert.Equal(t, NodeTaskReasonTimeout, nodeStatus[0].Reason)
		})
	}

	t.Run("only unfinished node tasks are timed out", func(t *testing.T) {
		job := newJob(operationsv1alpha2.JobPhaseInProgress, &timeout, expired,
			nodeTask(operationsv1alpha2.NodeTaskPhaseSuccessful, expired.Format(time.RFC3339)),
			nodeTask(operationsv1alpha2.NodeTaskPhaseInProgress, expired.Format(time.RFC3339)))
		cli := fakeConfigUpdateJobClient(job)
		handler := NewConfigUpdateJobReconcileHandler(cli, nil)

		require.NoError(t, handler.CheckTimeout(ctx, jobName))
		nodeStatus := getNodeStatus(t, cli)
		require.Len(t, nodeStatus, 2)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhaseSuccessful, nodeStatus[0].Phase)
		assert.Empty(t, nodeStatus[0].Reason)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhaseUnknown, nodeStatus[1].Phase)
		assert.Equal(t, NodeTaskReasonTimeout, nodeStatus[1].Reason)
	})

	t.Run("invalid last action time", func(t *testing.T) {
		job := newJob(operationsv1alpha2.JobPhaseInProgress, &timeout, expired,
			nodeTask(operationsv1alpha2.NodeTaskPhaseInProgress, "invalid-time"))
		cli := fakeConfigUpdateJobClient(job)
		handler := NewConfigUpdateJobReconcileHandler(cli, nil)

		err := handler.CheckTimeout(ctx, jobName)
		assert.ErrorContains(t, err, "failed to parse last action update time invalid-time")
	})

	t.Run("failed to update job status", func(t *testing.T) {
		job := newJob(operationsv1alpha2.JobPhaseInProgress, &timeout, expired,
			nodeTask(operationsv1alpha2.NodeTaskPhasePending))
		cli := interceptor.NewClient(fakeConfigUpdateJobClient(job).(client.WithWatch), interceptor.Funcs{
			SubResourceUpdate: func(_ctx context.Context, _cli client.Client, _subResource string,
				_obj client.Object, _opts ...client.SubResourceUpdateOption,
			) error {
				return errors.New("test error")
			},
		})
		handler := NewConfigUpdateJobReconcileHandler(cli, nil)

		err := handler.CheckTimeout(ctx, jobName)
		assert.ErrorContains(t, err, "failed to update configupdate job test-job status")
	})
}

func fakeConfigUpdateJobClient(objs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(operationsv1alpha2.SchemeGroupVersion,
		&operationsv1alpha2.ConfigUpdateJob{},
		&operationsv1alpha2.ConfigUpdateJobList{},
	)
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&operationsv1alpha2.ConfigUpdateJob{}).
		Build()
}
