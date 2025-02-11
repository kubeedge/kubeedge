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
	RetryAttempts = 10
	RetryDelay    = 1 * time.Second
)

type TaskStatus interface {
	operationsv1alpha2.ImagePrePullNodeTaskStatus |
		operationsv1alpha2.NodeUpgradeJobNodeTaskStatus
}

type UpdateStatusOptions[T TaskStatus] struct {
	JobName string

	NodeTaskStatus T

	Callback func(err error)
}

type TryUpdateFun[T TaskStatus] func(ctx context.Context, cli crdcliset.Interface, opts UpdateStatusOptions[T]) error

type StatusUpdater[T TaskStatus] struct {
	ctx          context.Context
	crdcli       crdcliset.Interface
	updateCh     chan UpdateStatusOptions[T]
	tryUpdateFun TryUpdateFun[T]
}

func NewStatusUpdater[T TaskStatus](
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

func (u *StatusUpdater[T]) UpdateStatus(opts UpdateStatusOptions[T]) {
	u.updateCh <- opts
}

func (u *StatusUpdater[T]) WatchUpdateChannel() {
	for {
		select {
		case <-u.ctx.Done():
			return
		case opts := <-u.updateCh:
			err := retry.Do(
				func() error {
					return u.tryUpdateFun(u.ctx, u.crdcli, opts)
				},
				retry.Delay(RetryAttempts),
				retry.Attempts(RetryAttempts),
				retry.DelayType(retry.FixedDelay))
			if opts.Callback != nil {
				opts.Callback(err)
			}
		}
	}
}
