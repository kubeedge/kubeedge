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
type UpdateStatusOptions struct {
	// Callback is the callback function of UpdateStatus(..). It will report the result of updating the status.
	Callback func(err error)

	TryUpdateStatusOptions
}

// TryUpdateStatusOptions defines options for function of try to update status.
type TryUpdateStatusOptions struct {
	// JobName is the name of the node job.
	JobName string
	// NodeName is the name of the edge node.
	NodeName string
	// Phase is the phase of the node task.
	Phase operationsv1alpha2.NodeTaskPhase
	// Reason is the failure reason of the node task.
	Reason string
	// ExtendInfo is the extended reporting information.
	ExtendInfo string
	// ActionStatus is the action status of the node task.
	// I.e., *ImagePrePullJobActionStatus or *NodeUpgradeJobActionStatus,
	// or the action status of other node jobs that must be a pointer structure.
	ActionStatus any
}

// TryUpdateFun defines the function type for updating the status.
type TryUpdateFun func(ctx context.Context, cli crdcliset.Interface, opts TryUpdateStatusOptions) error

// StatusUpdater defines the updater of the node task status.
type StatusUpdater struct {
	ctx       context.Context
	crdcli    crdcliset.Interface
	updateCh  chan UpdateStatusOptions
	tryUpdate TryUpdateFun
}

// NewStatusUpdater returns a new StatusUpdater.
func NewStatusUpdater(
	ctx context.Context,
	tryUpdate TryUpdateFun,
) *StatusUpdater {
	return &StatusUpdater{
		ctx:       ctx,
		crdcli:    client.GetCRDClient(),
		updateCh:  make(chan UpdateStatusOptions, 100),
		tryUpdate: tryUpdate,
	}
}

// UpdateStatus sends the UpdateStatusOptions to the channel.
// Must call the WatchUpdateChannel() method before calling this method.
func (u *StatusUpdater) UpdateStatus(opts UpdateStatusOptions) {
	u.updateCh <- opts
}

// WatchUpdateChannel watches the update channel and updates the status of the node task.
// It will retry the update operation if the update fails.
func (u *StatusUpdater) WatchUpdateChannel() {
	for {
		select {
		case <-u.ctx.Done():
			return
		case opts := <-u.updateCh:
			err := retry.Do(
				func() error {
					return u.tryUpdate(u.ctx, u.crdcli, opts.TryUpdateStatusOptions)
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
