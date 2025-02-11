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

	typeoperationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	clientoperationsv1alpha2 "github.com/kubeedge/api/client/informers/externalversions/operations/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
)

// Check NodeUpgradeJobHandler implements Handler interface
var _ Handler = (*NodeUpgradeJobHandler)(nil)

type NodeUpgradeJobHandler struct {
	nodeUpgradeJobInformer clientoperationsv1alpha2.NodeUpgradeJobInformer
}

func NewNodeUpgradeJobHandler() *NodeUpgradeJobHandler {
	return &NodeUpgradeJobHandler{
		nodeUpgradeJobInformer: informers.GetInformersManager().
			GetKubeEdgeInformerFactory().
			Operations().
			V1alpha2().
			NodeUpgradeJobs(),
	}
}

func (NodeUpgradeJobHandler) Name() string {
	return typeoperationsv1alpha2.ResourceNodeUpgradeJob
}

func (h *NodeUpgradeJobHandler) Informer() cache.SharedIndexInformer {
	return h.nodeUpgradeJobInformer.Informer()
}

func (h *NodeUpgradeJobHandler) UpdateNodeActionStatus(ctx context.Context, msg model.Message) error {
	// TODO: ...
	return nil
}

func (h *NodeUpgradeJobHandler) OnAdd(obj any, isInInitialList bool) {
	nodeUpgradeJob, ok := obj.(*typeoperationsv1alpha2.NodeUpgradeJob)
	if !ok {
		klog.Errorf("Failed to convert obj to NodeUpgradeJob, obj type is: %T", obj)
		return
	}
	if isInInitialList && nodeUpgradeJob.Status.State.IsFinal() {
		klog.V(5).Infof("this node upgrade task '%s' is already in the final state, ignore it",
			nodeUpgradeJob.Name)
		return
	}
	h.downstreamNodeAction(nodeUpgradeJob)
}

func (h *NodeUpgradeJobHandler) OnUpdate(_oldObj, newObj any) {
	nodeUpgradeJob, ok := newObj.(*typeoperationsv1alpha2.NodeUpgradeJob)
	if !ok {
		klog.Errorf("Failed to convert obj to NodeUpgradeJob, obj type is: %T", newObj)
		return
	}
	h.downstreamNodeAction(nodeUpgradeJob)
}

func (h *NodeUpgradeJobHandler) downstreamNodeAction(job *typeoperationsv1alpha2.NodeUpgradeJob) {
	if job.Status.State != typeoperationsv1alpha2.JobStateInit &&
		job.Status.State != typeoperationsv1alpha2.JobStateInProgress {
		klog.Infof("node upgrade job %s is not in the init or in progress state, ignore it", job.Name)
		return
	}
	// TODO: ...
}

func (h *NodeUpgradeJobHandler) OnDelete(obj any) {
	nodeUpgradeJob, ok := obj.(*typeoperationsv1alpha2.NodeUpgradeJob)
	if !ok {
		klog.Errorf("Failed to convert obj to NodeUpgradeJob, obj type is: %T", obj)
		return
	}
	// TODO: Interrupt tasks
	klog.Infof("delete node upgrade job %v", nodeUpgradeJob)
}
