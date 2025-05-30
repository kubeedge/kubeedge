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

	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/commons"
)

type NodeUpgradeJobController struct {
	handler ReconcileHandler[operationsv1alpha2.NodeUpgradeJob]
}

func NewNodeUpgradeJobController(cli client.Client, che cache.Cache,
) *NodeUpgradeJobController {
	return &NodeUpgradeJobController{
		handler: NewNodeUpgradeJobReconcileHandler(cli, che),
	}
}

// Reconcile reconciles the node upgrade job.
func (c *NodeUpgradeJobController) Reconcile(ctx context.Context, req controllerruntime.Request,
) (res controllerruntime.Result, resErr error) {
	logger := klog.FromContext(ctx).
		WithName(commons.LoggerNameNodeUpgradeJob).
		WithValues(commons.LoggerFieldInstanceName, req.Name,
			commons.LoggerFieldNodeJobType, operationsv1alpha2.ResourceNodeUpgradeJob)
	ctx = klog.NewContext(ctx, logger)

	logger.V(2).Info("reconciling the node upgrade job")

	defer func() {
		if e := recover(); e != nil {
			logger.Error(e.(error), "reconcile panic an error, do requeue", "stack", debug.Stack())
			res = controllerruntime.Result{RequeueAfter: commons.DefaultRequeueTime}
		}
	}()

	if err := RunReconcile(ctx, req, c.handler); err != nil {
		logger.Error(err, "failed to run reconcile, do requeue")
		return controllerruntime.Result{RequeueAfter: commons.DefaultRequeueTime}, nil
	}
	return
}

func (c *NodeUpgradeJobController) SetupWithManager(_ctx context.Context, mgr controllerruntime.Manager) error {
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&operationsv1alpha2.NodeUpgradeJob{}).
		Complete(c)
}
