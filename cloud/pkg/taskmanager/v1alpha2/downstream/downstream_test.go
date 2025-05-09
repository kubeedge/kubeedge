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
	"context"
	"sync"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/session"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/executor"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/wrap"
)

func TestWatchJobDownstream(t *testing.T) {
	t.Run("executor created, execution ignored", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		logger := klog.Background()
		var wg sync.WaitGroup
		wg.Add(1)

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(executor.NewNodeTaskExecutor, func(
			_ctx context.Context,
			_job wrap.NodeJob,
			_updateFun executor.UpdateNodeTaskStatus,
		) (*executor.NodeTaskExecutor, bool, error) {
			return &executor.NodeTaskExecutor{}, true, nil
		})
		// Indicates that the print ignore log is called.
		patches.ApplyMethodFunc(logr.Logger{}, "Info", func(msg string, _keysAndValues ...any) {
			assert.Equal(t, "node task executor is already running, ignore it", msg)
			wg.Done()
		})

		downChan := make(chan wrap.NodeJob)
		handler := &ImagePrePullJobHandler{
			logger:         logger,
			downstreamChan: downChan,
		}
		go watchJobDownstream(ctx, handler)

		downChan <- wrap.ImagePrePullJob{
			Obj: &operationsv1alpha2.ImagePrePullJob{},
		}
		wg.Wait()
	})

	t.Run("run executor successful", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(executor.NewNodeTaskExecutor, func(
			_ctx context.Context,
			_job wrap.NodeJob,
			_updateFun executor.UpdateNodeTaskStatus,
		) (*executor.NodeTaskExecutor, bool, error) {
			return &executor.NodeTaskExecutor{}, false, nil
		})
		patches.ApplyFunc(cloudhub.GetSessionManager, func() (*session.Manager, error) {
			return &session.Manager{}, nil
		})
		// Indicates that execute function is called.
		patches.ApplyMethodFunc(&executor.NodeTaskExecutor{}, "Execute",
			func(_ctx context.Context, _connectedNodes []string) {
				wg.Done()
			})

		downChan := make(chan wrap.NodeJob)
		handler := &ImagePrePullJobHandler{
			logger:         klog.Background(),
			downstreamChan: downChan,
		}
		go watchJobDownstream(ctx, handler)

		downChan <- wrap.ImagePrePullJob{
			Obj: &operationsv1alpha2.ImagePrePullJob{},
		}
		wg.Wait()
	})
}
