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
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/executor"
)

func TestNodeUpgradeJobCanDownstreamPhase(t *testing.T) {
	cases := []struct {
		name string
		obj  any
		want bool
	}{
		{
			name: "invalid obj type",
			obj:  operationsv1alpha2.NodeUpgradeJobList{},
			want: false,
		},
		{
			name: "cannot downstream phase",
			obj: &operationsv1alpha2.NodeUpgradeJob{
				Status: operationsv1alpha2.NodeUpgradeJobStatus{
					Phase: operationsv1alpha2.JobPhaseInProgress,
				},
			},
			want: false,
		},
		{
			name: "can downstream phase",
			obj: &operationsv1alpha2.NodeUpgradeJob{
				Status: operationsv1alpha2.NodeUpgradeJobStatus{
					Phase: operationsv1alpha2.JobPhaseInit,
				},
			},
			want: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			handler := &NodeUpgradeJobHandler{
				logger: klog.Background(),
			}
			assert.Equal(t, c.want, handler.CanDownstreamPhase(c.obj))
		})
	}
}

func TestNodeUpgradeJobInterruptExecutor(t *testing.T) {
	var interrupted, removed bool

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(executor.GetExecutor, func(_resourceType, _jobname string,
	) (*executor.NodeTaskExecutor, error) {
		return &executor.NodeTaskExecutor{}, nil
	})
	patches.ApplyMethodFunc(&executor.NodeTaskExecutor{}, "Interrupt", func() {
		interrupted = true
	})
	patches.ApplyFunc(executor.RemoveExecutor, func(_resourceType, _jobName string) {
		removed = true
	})

	handler := &NodeUpgradeJobHandler{
		logger: klog.Background(),
	}
	handler.InterruptExecutor(&operationsv1alpha2.NodeUpgradeJob{})
	assert.True(t, interrupted)
	assert.True(t, removed)
}
