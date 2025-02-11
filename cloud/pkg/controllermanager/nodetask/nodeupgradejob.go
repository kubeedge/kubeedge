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
	"runtime/debug"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/commons"
)

type NodeUpgradeJobController struct {
	cli client.Client
	che cache.Cache
}

func NewNodeUpgradeJobController(cli client.Client, che cache.Cache,
) *NodeUpgradeJobController {
	return &NodeUpgradeJobController{
		cli: cli,
	}
}

func (c *NodeUpgradeJobController) Reconcile(ctx context.Context, req controllerruntime.Request,
) (res controllerruntime.Result, resErr error) {
	logger := klog.FromContext(ctx).
		WithName(commons.LoggerNameNodeUpgradeJob).
		WithValues(commons.LoggerFieldInstanceName, req.Name)
	ctx = klog.NewContext(ctx, logger)

	logger.V(1).Info("reconciling the node upgrade job")

	defer func() {
		if e := recover(); e != nil {
			logger.Error(e.(error), "reconcile panic an error, do requeue", "stack", debug.Stack())
			res = controllerruntime.Result{RequeueAfter: commons.DefaultRequeueTime}
		}
	}()

	var job operationsv1alpha2.NodeUpgradeJob
	if err := c.cli.Get(ctx, req.NamespacedName, &job); err != nil {
		// The resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		logger.Error(err, "failed to get job, do requeue")
		return controllerruntime.Result{RequeueAfter: commons.DefaultRequeueTime}, nil
	}

	var changed bool
	if !c.HasInitialized(&job) {
		b, err := c.InitNodesStatus(&job)
		if err != nil {
			logger.Error(err, "failed to init nodes status, do requeue")
			return controllerruntime.Result{RequeueAfter: commons.DefaultRequeueTime}, nil
		}
		changed = b
	} else {
		b, err := c.CalculateStatus(&job)
		if err != nil {
			logger.Error(err, "failed to calculate nodes status, do requeue")
			return controllerruntime.Result{RequeueAfter: commons.DefaultRequeueTime}, nil
		}
		changed = b
	}
	if changed {
		// TODO: update
	}
	return
}

func (c *NodeUpgradeJobController) SetupWithManager(_ctx context.Context, mgr controllerruntime.Manager) error {
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&operationsv1alpha2.NodeUpgradeJob{}).
		Complete(c)
}
