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
	"time"

	retry "github.com/avast/retry-go"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	crdcliset "github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
)

const (
	UpdateStatusRetryAttempts = 5
	UpdateStatusRetryDelay    = 200 * time.Millisecond
)

// UpdateStatusOptions defines options for update status.
type UpdateStatusOptions[T operationsv1alpha2.NodeTaskStatusType] struct {
	// JobName is the name of the node job.
	JobName string
	// NodeTaskStatus is the node task status object.
	NodeTaskStatus T
	// Callback is the callback function of UpdateStatus(..). It will report the result of updating the status.
	Callback func(err error)
}

// TryUpdateFun defines the function type for updateing the status.
type TryUpdateFun[T operationsv1alpha2.NodeTaskStatusType] func(
	ctx context.Context,
	cli crdcliset.Interface,
	jobName string,
	nodeTaskStatus T) error

// StatusUpdater defines the updater of the node task status.
type StatusUpdater[T operationsv1alpha2.NodeTaskStatusType] struct {
	ctx          context.Context
	crdcli       crdcliset.Interface
	updateCh     chan UpdateStatusOptions[T]
	tryUpdateFun TryUpdateFun[T]
}

// NewStatusUpdater returns a new StatusUpdater.
func NewStatusUpdater[T operationsv1alpha2.NodeTaskStatusType](
	ctx context.Context,
	tryUpdateFun TryUpdateFun[T],
) *StatusUpdater[T] {
	return &StatusUpdater[T]{
		ctx:          ctx,
		crdcli:       client.GetCRDClient(),
		updateCh:     make(chan UpdateStatusOptions[T], 100),
		tryUpdateFun: tryUpdateFun,
	}
}

// UpdateStatus sends the UpdateStatusOptions to the channel.
// Must call the WatchUpdateChannel() method before calling this method.
func (u *StatusUpdater[T]) UpdateStatus(opts UpdateStatusOptions[T]) {
	u.updateCh <- opts
}

// WatchUpdateChannel watches the update channel and updates the status of the node task.
// It will retry the update operation if the update fails.
func (u *StatusUpdater[T]) WatchUpdateChannel() {
	for {
		select {
		case <-u.ctx.Done():
			return
		case opts := <-u.updateCh:
			err := retry.Do(
				func() error {
					return u.tryUpdateFun(u.ctx, u.crdcli, opts.JobName, opts.NodeTaskStatus)
				},
				retry.Delay(UpdateStatusRetryDelay),
				retry.Attempts(UpdateStatusRetryAttempts),
				retry.DelayType(retry.FixedDelay))
			if opts.Callback != nil {
				opts.Callback(err)
			}
		}
	}
}
