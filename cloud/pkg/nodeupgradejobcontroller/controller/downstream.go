/*
Copyright 2022 The KubeEdge Authors.

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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryType "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/nodeupgradejobcontroller/manager"
	"github.com/kubeedge/kubeedge/common/constants"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	crdinformers "github.com/kubeedge/kubeedge/pkg/client/informers/externalversions"
)

type DownstreamController struct {
	kubeClient   kubernetes.Interface
	informer     k8sinformer.SharedInformerFactory
	crdClient    crdClientset.Interface
	messageLayer messagelayer.MessageLayer

	nodeUpgradeJobManager *manager.NodeUpgradeJobManager
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	klog.Info("Start NodeUpgradeJob Downstream Controller")

	go dc.syncNodeUpgradeJob()

	return nil
}

// syncNodeUpgradeJob is used to get events from informer
func (dc *DownstreamController) syncNodeUpgradeJob() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop sync NodeUpgradeJob")
			return
		case e := <-dc.nodeUpgradeJobManager.Events():
			upgrade, ok := e.Object.(*v1alpha1.NodeUpgradeJob)
			if !ok {
				klog.Warningf("object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				dc.nodeUpgradeJobAdded(upgrade)
			case watch.Deleted:
				dc.nodeUpgradeJobDeleted(upgrade)
			case watch.Modified:
				dc.nodeUpgradeJobUpdated(upgrade)
			default:
				klog.Warningf("NodeUpgradeJob event type: %s unsupported", e.Type)
			}
		}
	}
}

func buildUpgradeResource(upgradeID, nodeID string) string {
	resource := fmt.Sprintf("%s%s%s%s%s%s%s", NodeUpgrade, constants.ResourceSep, upgradeID, constants.ResourceSep, "node", constants.ResourceSep, nodeID)
	return resource
}

// nodeUpgradeJobAdded is used to process addition of new NodeUpgradeJob in apiserver
func (dc *DownstreamController) nodeUpgradeJobAdded(upgrade *v1alpha1.NodeUpgradeJob) {
	klog.V(4).Infof("add NodeUpgradeJob: %v", upgrade)
	// store in cache map
	dc.nodeUpgradeJobManager.UpgradeMap.Store(upgrade.Name, upgrade)

	// If all or partial edge nodes upgrade is upgrading or completed, we don't need to send upgrade message
	if isCompleted(upgrade) {
		klog.Errorf("The nodeUpgradeJob is already running or completed, don't send upgrade message again")
		return
	}

	// get node list that need upgrading
	var nodesToUpgrade []string
	if len(upgrade.Spec.NodeNames) != 0 {
		for _, node := range upgrade.Spec.NodeNames {
			nodeInfo, err := dc.informer.Core().V1().Nodes().Lister().Get(node)
			if err != nil {
				klog.Errorf("Failed to get node(%s) info: %v", node, err)
				continue
			}

			if needUpgrade(nodeInfo, upgrade.Spec.Version) {
				nodesToUpgrade = append(nodesToUpgrade, nodeInfo.Name)
			}
		}
	} else if upgrade.Spec.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(upgrade.Spec.LabelSelector)
		if err != nil {
			klog.Errorf("LabelSelector(%s) is not valid: %v", upgrade.Spec.LabelSelector, err)
			return
		}

		nodes, err := dc.informer.Core().V1().Nodes().Lister().List(selector)
		if err != nil {
			klog.Errorf("Failed to get nodes with label %s: %v", selector.String(), err)
			return
		}

		for _, node := range nodes {
			if needUpgrade(node, upgrade.Spec.Version) {
				nodesToUpgrade = append(nodesToUpgrade, node.Name)
			}
		}
	}

	// deduplicate: remove duplicate nodes to avoid repeating upgrade to the same node
	nodesToUpgrade = RemoveDuplicateElement(nodesToUpgrade)

	klog.Infof("Filtered finished, the below nodes are to upgrade\n%v\n", nodesToUpgrade)

	for _, node := range nodesToUpgrade {
		// send upgrade msg to every edge node
		msg := model.NewMessage("")

		resource := buildUpgradeResource(upgrade.Name, node)

		upgradeReq := commontypes.NodeUpgradeJobRequest{
			UpgradeID:   upgrade.Name,
			HistoryID:   uuid.New().String(),
			UpgradeTool: upgrade.Spec.UpgradeTool,
			Version:     upgrade.Spec.Version,
			Image:       upgrade.Spec.Image,
		}

		msg.BuildRouter(modules.NodeUpgradeJobControllerModuleName, modules.NodeUpgradeJobControllerModuleGroup, resource, NodeUpgrade).
			FillBody(upgradeReq)

		err := dc.messageLayer.Send(*msg)
		if err != nil {
			klog.Errorf("Failed to send upgrade message %v due to error %v", msg.GetID(), err)
			continue
		}

		// process time out: cloud did not receive upgrade feedback from edge
		// send upgrade timeout response message to upstream
		go dc.handleNodeUpgradeJobTimeout(node, upgrade.Name, upgrade.Spec.Version, upgradeReq.HistoryID, upgrade.Spec.TimeoutSeconds)

		// mark Upgrade state upgrading
		status := &v1alpha1.UpgradeStatus{
			NodeName: node,
			State:    v1alpha1.Upgrading,
			History: v1alpha1.History{
				HistoryID:   upgradeReq.HistoryID,
				UpgradeTime: time.Now().String(),
			},
		}
		err = patchNodeUpgradeJobStatus(dc.crdClient, upgrade, status)
		if err != nil {
			// not return, continue to mark node unschedulable
			klog.Errorf("Failed to mark Upgrade upgrading status: %v", err)
		}

		// mark edge node unschedulable
		// the effect is like running cmd: kubectl drain <node-to-drain> --ignore-daemonsets
		unscheduleNode := v1.Node{}
		unscheduleNode.Spec.Unschedulable = true
		// add a upgrade label
		unscheduleNode.Labels = map[string]string{NodeUpgradeJobStatusKey: NodeUpgradeJobStatusValue}
		byteNode, err := json.Marshal(unscheduleNode)
		if err != nil {
			klog.Warningf("marshal data failed: %v", err)
			continue
		}

		_, err = dc.kubeClient.CoreV1().Nodes().Patch(context.Background(), node, apimachineryType.StrategicMergePatchType, byteNode, metav1.PatchOptions{})
		if err != nil {
			klog.Errorf("failed to drain node %s: %v", node, err)
			continue
		}
	}
}

func needUpgrade(node *v1.Node, upgradeVersion string) bool {
	if filterVersion(node.Status.NodeInfo.KubeletVersion, upgradeVersion) {
		klog.Warningf("Node(%s) version(%s) already on the expected version %s.", node.Name, node.Status.NodeInfo.KubeletVersion, upgradeVersion)
		return false
	}

	// we only care about edge nodes, so just remove not edge nodes
	if !isEdgeNode(node) {
		klog.Warningf("Node(%s) is not edge node", node.Name)
		return false
	}

	// if node is in Upgrading state, don't need upgrade
	if _, ok := node.Labels[NodeUpgradeJobStatusKey]; ok {
		klog.Warningf("Node(%s) is in upgrade state", node.Name)
		return false
	}

	// if node is in NotReady state, don't need upgrade
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady && condition.Status != v1.ConditionTrue {
			klog.Warningf("Node(%s) is in NotReady state", node.Name)
			return false
		}
	}

	return true
}

// handleNodeUpgradeJobTimeout is used to handle the situation that cloud don't receive upgrade result from edge node
// within the timeout period
func (dc *DownstreamController) handleNodeUpgradeJobTimeout(node string, upgradeID string, upgradeVersion string, historyID string, timeoutSeconds *uint32) {
	// by default, if we don't receive upgrade response in 300s, we think it's timeout
	// if we have specified the timeout in Upgrade, we'll use it as the timeout time
	var timeout uint32 = 300
	if timeoutSeconds != nil && *timeoutSeconds != 0 {
		timeout = *timeoutSeconds
	}

	receiveFeedback := false

	// check whether edgecore report to the cloud about the upgrade result
	// we don't care about function Poll return error
	// if we don't receive Upgrade response, Poll function also return error: timed out waiting for the condition
	// we only care about variable: receiveFeedback
	_ = wait.Poll(10*time.Second, time.Duration(timeout)*time.Second, func() (bool, error) {
		v, ok := dc.nodeUpgradeJobManager.UpgradeMap.Load(upgradeID)
		if !ok {
			// we think it's receiveFeedback to avoid construct timeout response by ourselves
			receiveFeedback = true
			klog.Errorf("NodeUpgradeJob %v not exist", upgradeID)
			return false, fmt.Errorf("nodeUpgrade %v not exist", upgradeID)
		}
		upgradeValue := v.(*v1alpha1.NodeUpgradeJob)
		for index := range upgradeValue.Status.Status {
			if upgradeValue.Status.Status[index].NodeName == node {
				// if HistoryID matches and state is v1alpha1.Completed
				// it means we've received the specified Upgrade Operation
				if upgradeValue.Status.Status[index].History.HistoryID == historyID &&
					upgradeValue.Status.Status[index].State == v1alpha1.Completed {
					receiveFeedback = true
					return true, nil
				}
				break
			}
		}
		return false, nil
	})

	if receiveFeedback {
		// if already receive edge upgrade feedback, do nothing
		return
	}

	klog.Errorf("NOT receive node(%s) upgrade(%s) feedback response", node, upgradeID)

	// construct timeout upgrade response
	// and send it to upgrade controller upstream
	upgradeResource := buildUpgradeResource(upgradeID, node)
	resp := commontypes.NodeUpgradeJobResponse{
		UpgradeID:   upgradeID,
		HistoryID:   historyID,
		NodeName:    node,
		FromVersion: "",
		ToVersion:   upgradeVersion,
		Status:      string(v1alpha1.UpgradeFailedRollbackSuccess),
		Reason:      "timeout to get upgrade response from edge, maybe error due to cloud or edge",
	}

	updateMsg := model.NewMessage("").
		BuildRouter(modules.NodeUpgradeJobControllerModuleName, modules.NodeUpgradeJobControllerModuleGroup, upgradeResource, NodeUpgrade).
		FillBody(resp)

	// send upgrade resp message to upgrade controller upstream directly
	// let upgrade controller upstream to update upgrade status
	beehiveContext.Send(modules.NodeUpgradeJobControllerModuleName, *updateMsg)
}

// nodeUpgradeJobDeleted is used to process deleted NodeUpgradeJob in apiserver
func (dc *DownstreamController) nodeUpgradeJobDeleted(upgrade *v1alpha1.NodeUpgradeJob) {
	// just need to delete from cache map
	dc.nodeUpgradeJobManager.UpgradeMap.Delete(upgrade.Name)
}

// upgradeAdded is used to process update of new NodeUpgradeJob in apiserver
func (dc *DownstreamController) nodeUpgradeJobUpdated(upgrade *v1alpha1.NodeUpgradeJob) {
	oldValue, ok := dc.nodeUpgradeJobManager.UpgradeMap.Load(upgrade.Name)
	// store in cache map
	dc.nodeUpgradeJobManager.UpgradeMap.Store(upgrade.Name, upgrade)
	if !ok {
		klog.Infof("Upgrade %s not exist, and store it first", upgrade.Name)
		// If Upgrade not present in Upgrade map means it is not modified and added.
		dc.nodeUpgradeJobAdded(upgrade)
		return
	}

	old := oldValue.(*v1alpha1.NodeUpgradeJob)
	if !isUpgradeUpdated(upgrade, old) {
		klog.V(4).Infof("Upgrade %s no need to update", upgrade.Name)
		return
	}

	dc.nodeUpgradeJobAdded(upgrade)
}

// isUpgradeUpdated checks Upgrade is actually updated or not
func isUpgradeUpdated(new *v1alpha1.NodeUpgradeJob, old *v1alpha1.NodeUpgradeJob) bool {
	// now we don't allow update spec fields
	// so always return false to avoid sending Upgrade msg to edge again when status fields changed
	return false
}

func NewDownstreamController(crdInformerFactory crdinformers.SharedInformerFactory) (*DownstreamController, error) {
	nodeUpgradeJobManager, err := manager.NewNodeUpgradeJobManager(crdInformerFactory.Operations().V1alpha1().NodeUpgradeJobs().Informer())
	if err != nil {
		klog.Warningf("Create NodeUpgradeJob manager failed with error: %s", err)
		return nil, err
	}

	dc := &DownstreamController{
		kubeClient:            client.GetKubeClient(),
		informer:              informers.GetInformersManager().GetK8sInformerFactory(),
		crdClient:             client.GetCRDClient(),
		nodeUpgradeJobManager: nodeUpgradeJobManager,
		messageLayer:          messagelayer.NodeUpgradeJobControllerMessageLayer(),
	}
	return dc, nil
}
