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

package upstream

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	cloudcoreConfig "github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	crdClientset "github.com/kubeedge/api/client/clientset/versioned"
	operationsv1alpha2client "github.com/kubeedge/api/client/clientset/versioned/typed/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/status"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type mockOperations struct {
	operationsv1alpha2client.OperationsV1alpha2Interface
	imagePrePullJobs operationsv1alpha2client.ImagePrePullJobInterface
}

func (m *mockOperations) ImagePrePullJobs() operationsv1alpha2client.ImagePrePullJobInterface {
	return m.imagePrePullJobs
}

type mockImagePrePullJobs struct {
	operationsv1alpha2client.ImagePrePullJobInterface
	getFunc    func(ctx context.Context, name string, opts metav1.GetOptions) (*operationsv1alpha2.ImagePrePullJob, error)
	updateFunc func(ctx context.Context, job *operationsv1alpha2.ImagePrePullJob, opts metav1.UpdateOptions) (*operationsv1alpha2.ImagePrePullJob, error)
}

func (m *mockImagePrePullJobs) Get(ctx context.Context, name string, opts metav1.GetOptions) (*operationsv1alpha2.ImagePrePullJob, error) {
	return m.getFunc(ctx, name, opts)
}

func (m *mockImagePrePullJobs) UpdateStatus(ctx context.Context, job *operationsv1alpha2.ImagePrePullJob, opts metav1.UpdateOptions) (*operationsv1alpha2.ImagePrePullJob, error) {
	return m.updateFunc(ctx, job, opts)
}

func TestImagePrePullJobUpdateNodeTaskStatus(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(kubernetes.NewForConfigAndClient, func(c *rest.Config, httpClient *http.Client) (*kubernetes.Clientset, error) {
		return &kubernetes.Clientset{}, nil
	})
	patches.ApplyFunc(dynamic.NewForConfigAndClient, func(c *rest.Config, httpClient *http.Client) (*dynamic.DynamicClient, error) {
		return &dynamic.DynamicClient{}, nil
	})
	patches.ApplyFunc(crdClientset.NewForConfigAndClient, func(c *rest.Config, httpClient *http.Client) (*crdClientset.Clientset, error) {
		return &crdClientset.Clientset{}, nil
	})

	config := &cloudcoreConfig.KubeAPIConfig{
		Master: "http://localhost:8080",
	}
	client.InitKubeEdgeClient(config, false)

	status.Init(context.TODO())

	var (
		jobName  = "test-job"
		nodeNmae = "node1"
	)

	t.Run("final action successful", func(t *testing.T) {
		subPatches := gomonkey.NewPatches()
		defer subPatches.Reset()

		var called bool
		mockJobs := &mockImagePrePullJobs{
			getFunc: func(ctx context.Context, name string, opts metav1.GetOptions) (*operationsv1alpha2.ImagePrePullJob, error) {
				return &operationsv1alpha2.ImagePrePullJob{
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
					},
					Status: operationsv1alpha2.ImagePrePullJobStatus{
						NodeStatus: []operationsv1alpha2.ImagePrePullNodeTaskStatus{
							{
								NodeName: nodeNmae,
							},
						},
					},
				}, nil
			},
			updateFunc: func(ctx context.Context, job *operationsv1alpha2.ImagePrePullJob, opts metav1.UpdateOptions) (*operationsv1alpha2.ImagePrePullJob, error) {
				assert.Equal(t, jobName, job.Name)
				require.Len(t, job.Status.NodeStatus, 1)
				nodeStatus := job.Status.NodeStatus[0]
				assert.Equal(t, nodeNmae, nodeStatus.NodeName)
				assert.Equal(t, operationsv1alpha2.NodeTaskPhaseSuccessful, nodeStatus.Phase)
				require.Len(t, nodeStatus.ActionFlow, 1)
				assert.Equal(t, operationsv1alpha2.ImagePrePullJobActionPull, nodeStatus.ActionFlow[0].Action)
				called = true
				return job, nil
			},
		}

		mockOps := &mockOperations{
			imagePrePullJobs: mockJobs,
		}

		var c *crdClientset.Clientset
		subPatches.ApplyMethod(reflect.TypeOf(c), "OperationsV1alpha2",
			func(_ *crdClientset.Clientset) operationsv1alpha2client.OperationsV1alpha2Interface {
				return mockOps
			})

		handler := &ImagePrePullJobHandler{}
		err := handler.UpdateNodeTaskStatus(jobName, nodeNmae, true, taskmsg.UpstreamMessage{
			Action: string(operationsv1alpha2.ImagePrePullJobActionPull),
			Succ:   true,
		})
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("final action failed", func(t *testing.T) {
		subPatches := gomonkey.NewPatches()
		defer subPatches.Reset()

		var called bool
		mockJobs := &mockImagePrePullJobs{
			getFunc: func(ctx context.Context, name string, opts metav1.GetOptions) (*operationsv1alpha2.ImagePrePullJob, error) {
				return &operationsv1alpha2.ImagePrePullJob{
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
					},
					Status: operationsv1alpha2.ImagePrePullJobStatus{
						NodeStatus: []operationsv1alpha2.ImagePrePullNodeTaskStatus{
							{
								NodeName: nodeNmae,
							},
						},
					},
				}, nil
			},
			updateFunc: func(ctx context.Context, job *operationsv1alpha2.ImagePrePullJob, opts metav1.UpdateOptions) (*operationsv1alpha2.ImagePrePullJob, error) {
				assert.Equal(t, jobName, job.Name)
				require.Len(t, job.Status.NodeStatus, 1)
				nodeStatus := job.Status.NodeStatus[0]
				assert.Equal(t, nodeNmae, nodeStatus.NodeName)
				assert.Equal(t, operationsv1alpha2.NodeTaskPhaseFailure, nodeStatus.Phase)
				require.Len(t, nodeStatus.ActionFlow, 1)
				assert.Equal(t, operationsv1alpha2.ImagePrePullJobActionPull, nodeStatus.ActionFlow[0].Action)
				called = true
				return job, nil
			},
		}

		mockOps := &mockOperations{
			imagePrePullJobs: mockJobs,
		}

		var c *crdClientset.Clientset
		subPatches.ApplyMethod(reflect.TypeOf(c), "OperationsV1alpha2",
			func(_ *crdClientset.Clientset) operationsv1alpha2client.OperationsV1alpha2Interface {
				return mockOps
			})

		handler := &ImagePrePullJobHandler{}
		err := handler.UpdateNodeTaskStatus(jobName, nodeNmae, true, taskmsg.UpstreamMessage{
			Action: string(operationsv1alpha2.ImagePrePullJobActionPull),
			Succ:   false,
		})
		require.NoError(t, err)
		assert.True(t, called)
	})
}
