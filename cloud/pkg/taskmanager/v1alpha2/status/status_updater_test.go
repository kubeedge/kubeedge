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

package status

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	crdcliset "github.com/kubeedge/api/client/clientset/versioned"
)

func TestUpdateStatus(t *testing.T) {
	t.Run("update status successful", func(t *testing.T) {
		var wg sync.WaitGroup
		var callback bool
		wg.Add(1)
		tryUpdateFun := func(_ctx context.Context,
			_cli crdcliset.Interface,
			_nodeName string,
			_nodeTaskStatus operationsv1alpha2.ImagePrePullNodeTaskStatus,
		) error {
			return nil
		}
		updater := NewStatusUpdater(context.Background(), tryUpdateFun)
		go updater.WatchUpdateChannel()
		time.Sleep(1 * time.Second)

		updater.UpdateStatus(UpdateStatusOptions[operationsv1alpha2.ImagePrePullNodeTaskStatus]{
			JobName:        "test",
			NodeTaskStatus: operationsv1alpha2.ImagePrePullNodeTaskStatus{},
			Callback: func(err error) {
				defer wg.Done()
				callback = true
				assert.NoError(t, err)
			},
		})
		wg.Wait()
		assert.True(t, callback)
	})

	t.Run("update status failed", func(t *testing.T) {
		var wg sync.WaitGroup
		var callback bool
		wg.Add(1)
		tryUpdateFun := func(_ctx context.Context,
			_cli crdcliset.Interface,
			_nodeName string,
			_nodeTaskStatus operationsv1alpha2.ImagePrePullNodeTaskStatus,
		) error {
			return errors.New("test error")
		}
		updater := NewStatusUpdater(context.Background(), tryUpdateFun)
		go updater.WatchUpdateChannel()
		time.Sleep(1 * time.Second)

		updater.UpdateStatus(UpdateStatusOptions[operationsv1alpha2.ImagePrePullNodeTaskStatus]{
			JobName:        "test",
			NodeTaskStatus: operationsv1alpha2.ImagePrePullNodeTaskStatus{},
			Callback: func(err error) {
				defer wg.Done()
				callback = true
				assert.Error(t, err)
				assert.ErrorContains(t, err, "test error")
			},
		})
		wg.Wait()
		assert.True(t, callback)
	})
}
