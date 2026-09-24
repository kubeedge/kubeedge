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

package downstream

import (
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/executor"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/wrap"
)

func TestNodeJobEventHandlerOnDeleteHandlesTombstone(t *testing.T) {
	var interrupted, removed bool

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	job := &operationsv1alpha2.ImagePrePullJob{}
	job.Name = "image-pre-pull-job"

	patches.ApplyFunc(executor.GetExecutor, func(resourceType, jobName string,
	) (*executor.NodeTaskExecutor, error) {
		assert.Equal(t, operationsv1alpha2.ResourceImagePrePullJob, resourceType)
		assert.Equal(t, job.Name, jobName)
		return &executor.NodeTaskExecutor{}, nil
	})
	patches.ApplyMethodFunc(&executor.NodeTaskExecutor{}, "Interrupt", func() {
		interrupted = true
	})
	patches.ApplyFunc(executor.RemoveExecutor, func(resourceType, jobName string) {
		assert.Equal(t, operationsv1alpha2.ResourceImagePrePullJob, resourceType)
		assert.Equal(t, job.Name, jobName)
		removed = true
	})

	originDownstreamHandlers := downstreamHandlers
	t.Cleanup(func() {
		downstreamHandlers = originDownstreamHandlers
	})
	downstreamHandlers = map[string]DownstreamHandler{
		operationsv1alpha2.ResourceImagePrePullJob: &ImagePrePullJobHandler{
			logger: klog.Background(),
		},
	}

	eventHandler := NewNodeJobEventHandler(klog.Background(), make(chan wrap.NodeJob))
	eventHandler.OnDelete(cache.DeletedFinalStateUnknown{Obj: job})

	assert.True(t, interrupted)
	assert.True(t, removed)
}
