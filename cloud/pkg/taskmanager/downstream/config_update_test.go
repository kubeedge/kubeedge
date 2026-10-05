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

package downstream

import (
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/executor"
)

func TestConfigUpdateJobCanDownstreamPhase(t *testing.T) {
	cases := []struct {
		name string
		obj  any
		want bool
	}{
		{
			name: "invalid obj type",
			obj:  operationsv1alpha2.ConfigUpdateJobList{},
			want: false,
		},
		{
			name: "cannot downstream phase",
			obj: &operationsv1alpha2.ConfigUpdateJob{
				Status: operationsv1alpha2.ConfigUpdateJobStatus{
					Phase: operationsv1alpha2.JobPhaseInProgress,
				},
			},
			want: false,
		},
		{
			name: "can downstream phase",
			obj: &operationsv1alpha2.ConfigUpdateJob{
				Status: operationsv1alpha2.ConfigUpdateJobStatus{
					Phase: operationsv1alpha2.JobPhaseInit,
				},
			},
			want: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			handler := &ConfigUpdateJobHandler{
				logger: logr.Discard(),
			}
			assert.Equal(t, c.want, handler.CanDownstreamPhase(c.obj))
		})
	}
}

func TestConfigUpdateJobInterruptExecutor(t *testing.T) {
	const jobName = "config-update-job"

	var interrupted, removed bool

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(executor.GetExecutor, func(resourceType, name string,
	) (*executor.NodeTaskExecutor, error) {
		assert.Equal(t, operationsv1alpha2.ResourceConfigUpdateJob, resourceType)
		assert.Equal(t, jobName, name)
		return &executor.NodeTaskExecutor{}, nil
	})
	patches.ApplyMethodFunc(&executor.NodeTaskExecutor{}, "Interrupt", func() {
		interrupted = true
	})
	patches.ApplyFunc(executor.RemoveExecutor, func(resourceType, name string) {
		assert.Equal(t, operationsv1alpha2.ResourceConfigUpdateJob, resourceType)
		assert.Equal(t, jobName, name)
		removed = true
	})

	handler := &ConfigUpdateJobHandler{
		logger: klog.Background(),
	}
	handler.InterruptExecutor(&operationsv1alpha2.ConfigUpdateJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
		},
	})
	assert.True(t, interrupted)
	assert.True(t, removed)
}
