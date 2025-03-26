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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
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
					Phase: operationsv1alpha2.JobPhaseComplated,
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
							BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
								NodeName: "node1",
								Phase:    operationsv1alpha2.NodeTaskPhaseInProgress,
							},
						},
						{
							BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
								NodeName: "node2",
								Phase:    operationsv1alpha2.NodeTaskPhaseSuccessful,
							},
						},
						{
							BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
								NodeName: "node3",
								Phase:    operationsv1alpha2.NodeTaskPhaseFailure,
							},
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
							BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
								NodeName: "node1",
								Phase:    operationsv1alpha2.NodeTaskPhaseUnknown,
							},
						},
						{
							BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
								NodeName: "node2",
								Phase:    operationsv1alpha2.NodeTaskPhaseSuccessful,
							},
						},
						{
							BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
								NodeName: "node3",
								Phase:    operationsv1alpha2.NodeTaskPhaseFailure,
							},
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
							BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
								NodeName: "node1",
								Phase:    operationsv1alpha2.NodeTaskPhaseSuccessful,
							},
						},
						{
							BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
								NodeName: "node2",
								Phase:    operationsv1alpha2.NodeTaskPhaseSuccessful,
							},
						},
						{
							BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
								NodeName: "node3",
								Phase:    operationsv1alpha2.NodeTaskPhaseFailure,
							},
						},
					},
				},
			},
			wantChanged: true,
			wantPhase:   operationsv1alpha2.JobPhaseComplated,
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
