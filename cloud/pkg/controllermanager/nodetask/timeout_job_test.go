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
	"github.com/stretchr/testify/require"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

func TestTimeoutJobManager(t *testing.T) {
	fakeHandler := &fakeReconcileHandler{}
	job, loaded := GetOrCreateTimeoutJob("test-job", fakeHandler.GetResource(), fakeHandler)
	t.Logf("created job: %p", job)
	assert.NotNil(t, job)
	assert.False(t, loaded)
	assert.Equal(t, "test-job", job.nodeJobName)

	val, ok := timeoutJobs.Load(timeoutJobsKey("test-job", fakeHandler.GetResource()))
	assert.True(t, ok)
	job, ok = val.(*TimeoutJob[operationsv1alpha2.ImagePrePullJob])
	require.True(t, ok)
	t.Logf("got job: %p", job)

	ReleaseTimeoutJob[operationsv1alpha2.ImagePrePullJob](context.TODO(), "test-job", fakeHandler.GetResource())
	assert.True(t, job.IsStopped())
	_, ok = timeoutJobs.Load(timeoutJobsKey("test-job", fakeHandler.GetResource()))
	assert.False(t, ok)
}

func TestTimeoutJob(t *testing.T) {
	ctx := context.TODO()

	t.Run("job has been stopped", func(t *testing.T) {
		fakeHandler := &fakeReconcileHandler{}
		job := NewTimeoutJob("test-job", fakeHandler)
		job.Stop(ctx)
		job.Run(ctx)
		assert.True(t, job.IsStopped())
		assert.Equal(t, 0, fakeHandler.called["CheckTimeout"])
	})

	t.Run("context is done", func(t *testing.T) {
		fakeHandler := &fakeReconcileHandler{}
		job := NewTimeoutJob("test-job", fakeHandler)
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		job.Run(ctx)
		assert.True(t, job.IsStopped())
		assert.Equal(t, 0, fakeHandler.called["CheckTimeout"])
	})

	t.Run("wait for call ChackTimeout method", func(t *testing.T) {
		fakeHandler := &fakeReconcileHandler{}
		job := &TimeoutJob[operationsv1alpha2.ImagePrePullJob]{
			nodeJobName: "test-job",
			handler:     fakeHandler,
			ticker:      time.NewTicker(1 * time.Second),
		}
		go job.Run(ctx)
		time.Sleep(1200 * time.Millisecond)
		job.Stop(ctx)
		assert.True(t, job.IsStopped())
		assert.Equal(t, 1, fakeHandler.called["CheckTimeout"])
	})
}
