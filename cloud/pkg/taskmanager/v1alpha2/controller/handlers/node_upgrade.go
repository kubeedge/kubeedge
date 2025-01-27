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

package handlers

import (
	"context"

	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
)

// Check NodeUpgradeJobHandler implements Handler interface
var _ Handler = (*NodeUpgradeJobHandler)(nil)

type NodeUpgradeJobHandler struct {
}

func NewNodeUpgradeJobHandler() *NodeUpgradeJobHandler {
	return &NodeUpgradeJobHandler{}
}

func (h *NodeUpgradeJobHandler) Name() string {
	return operationsv1alpha2.ResourceNodeUpgradeJob
}

func (NodeUpgradeJobHandler) Informer() cache.SharedIndexInformer {
	return informers.GetInformersManager().
		GetKubeEdgeInformerFactory().
		Operations().
		V1alpha2().
		NodeUpgradeJobs().
		Informer()
}

func (h *NodeUpgradeJobHandler) UpdateNodeActionStatus(ctx context.Context, msg model.Message) error {
	// TODO: ...
	return nil
}

func (h *NodeUpgradeJobHandler) OnAdd(obj any, _ bool) {
	// TODO: ...
	klog.Info("add node upgrade job %v", obj)
}

func (h *NodeUpgradeJobHandler) OnUpdate(old, new any) {
	// TODO: ...
	klog.Info("update node upgrade job %v", new)
}

func (h *NodeUpgradeJobHandler) OnDelete(obj any) {
	// TODO: ...
	klog.Info("delete node upgrade job %v", obj)
}
