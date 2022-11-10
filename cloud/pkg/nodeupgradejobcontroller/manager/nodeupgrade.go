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

package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryType "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	crdinformers "github.com/kubeedge/kubeedge/pkg/client/informers/externalversions"
)

// NodeUpgradeJobManager is a manager watch NodeUpgradeJob change event
type NodeUpgradeJobManager struct {
	// client
	kubeClient kubernetes.Interface
	crdClient  crdClientset.Interface

	// informer
	kubeInformer k8sinformer.SharedInformerFactory
	crdInformer  crdinformers.SharedInformerFactory

	messageLayer messagelayer.MessageLayer
}

var manager = &NodeUpgradeJobManager{}

func (m *NodeUpgradeJobManager) Start() {
	// filterNodeUpgradeJob nodeUpgradeJobAdded nodeUpgradeJobUpdated nodeUpgradeJobDeleted
	// will be executed automatically
	// So here do thing

}

// NewNodeUpgradeJobManager create NodeUpgradeJobManager from config
func NewNodeUpgradeJobManager() (*NodeUpgradeJobManager, error) {
	crdInformer := informers.GetInformersManager().GetKubeEdgeInformerFactory()
	si := crdInformer.Operations().V1alpha1().NodeUpgradeJobs().Informer()

	fh := NewFilterResourceEventHandler(filterNodeUpgradeJob, nodeUpgradeJobAdded, nodeUpgradeJobUpdated, nodeUpgradeJobDeleted)
	si.AddEventHandler(fh)

	manager = &NodeUpgradeJobManager{
		kubeClient:   client.GetKubeClient(),
		crdClient:    client.GetCRDClient(),
		kubeInformer: informers.GetInformersManager().GetKubeInformerFactory(),
		crdInformer:  crdInformer,
		messageLayer: messagelayer.NodeUpgradeJobControllerMessageLayer(),
	}
	return manager, nil
}

func filterNodeUpgradeJob(obj interface{}) bool {
	return true
}

func nodeUpgradeJobAdded(obj interface{}) {
	upgrade := obj.(*v1alpha1.NodeUpgradeJob)
	klog.V(4).Infof("add NodeUpgradeJob: %v", upgrade)

	// If all or partial edge nodes upgrade is upgrading or completed, we don't need to send upgrade message
	if isCompleted(upgrade) {
		klog.Errorf("The nodeUpgradeJob is already running or completed, don't send upgrade message again")
		return
	}

	// get node list that need upgrading
	var nodesToUpgrade []string
	if len(upgrade.Spec.NodeNames) != 0 {
		for _, node := range upgrade.Spec.NodeNames {
			nodeInfo, err := manager.kubeInformer.Core().V1().Nodes().Lister().Get(node)
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

		nodes, err := manager.kubeInformer.Core().V1().Nodes().Lister().List(selector)
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

	// if users specify Image, we'll use upgrade Version as its image tag, even though Image contains tag.
	// if not, we'll use default image: kubeedge/installation-package:${Version}
	var repo string
	var err error
	repo = "kubeedge/installation-package"
	if upgrade.Spec.Image != "" {
		repo, err = GetImageRepo(upgrade.Spec.Image)
		if err != nil {
			klog.Errorf("Image format is not right: %v", err)
			return
		}
	}
	imageTag := upgrade.Spec.Version
	image := fmt.Sprintf("%s:%s", repo, imageTag)

	for _, node := range nodesToUpgrade {
		// send upgrade msg to every edge node
		msg := model.NewMessage("")

		resource := buildUpgradeResource(upgrade.Name, node)

		upgradeReq := commontypes.NodeUpgradeJobRequest{
			UpgradeID:   upgrade.Name,
			HistoryID:   uuid.New().String(),
			UpgradeTool: upgrade.Spec.UpgradeTool,
			Version:     upgrade.Spec.Version,
			Image:       image,
		}

		msg.BuildRouter(modules.NodeUpgradeJobControllerModuleName, modules.NodeUpgradeJobControllerModuleGroup, resource, NodeUpgrade).
			FillBody(upgradeReq)

		err := manager.messageLayer.Send(*msg)
		if err != nil {
			klog.Errorf("Failed to send upgrade message %v due to error %v", msg.GetID(), err)
			continue
		}

		// process time out: cloud did not receive upgrade feedback from edge
		// send upgrade timeout response message to upstream
		go manager.handleNodeUpgradeJobTimeout(node, upgrade.Name, upgrade.Spec.Version, upgradeReq.HistoryID, upgrade.Spec.TimeoutSeconds)

		// mark Upgrade state upgrading
		status := &v1alpha1.UpgradeStatus{
			NodeName: node,
			State:    v1alpha1.Upgrading,
			History: v1alpha1.History{
				HistoryID:   upgradeReq.HistoryID,
				UpgradeTime: time.Now().String(),
			},
		}
		err = PatchNodeUpgradeJobStatus(manager.crdClient, upgrade, status)
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

		_, err = manager.kubeClient.CoreV1().Nodes().Patch(context.Background(), node, apimachineryType.StrategicMergePatchType, byteNode, metav1.PatchOptions{})
		if err != nil {
			klog.Errorf("failed to drain node %s: %v", node, err)
			continue
		}
	}
}

func nodeUpgradeJobDeleted(obj interface{}) {
	// do nothing
}

func nodeUpgradeJobUpdated(oldObj, newObj interface{}) {
	nodeUpgradeJobAdded(newObj)
}

func buildUpgradeResource(upgradeID, nodeID string) string {
	resource := fmt.Sprintf("%s%s%s%s%s%s%s", NodeUpgrade, constants.ResourceSep, upgradeID, constants.ResourceSep, "node", constants.ResourceSep, nodeID)
	return resource
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
func (m *NodeUpgradeJobManager) handleNodeUpgradeJobTimeout(node string, upgradeID string, upgradeVersion string, historyID string, timeoutSeconds *uint32) {
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
		upgradeValue, err := m.crdInformer.Operations().V1alpha1().NodeUpgradeJobs().Lister().Get(upgradeID)
		if err != nil {
			// we think it's receiveFeedback to avoid construct timeout response by ourselves
			receiveFeedback = true
			klog.Errorf("NodeUpgradeJob %v not exist", upgradeID)
			return false, fmt.Errorf("nodeUpgrade %v not exist", upgradeID)
		}

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

// PatchNodeUpgradeJobStatus call patch api to patch update NodeUpgradeJob status
func PatchNodeUpgradeJobStatus(crdClient crdClientset.Interface, upgrade *v1alpha1.NodeUpgradeJob, status *v1alpha1.UpgradeStatus) error {
	oldValue := upgrade.DeepCopy()

	newValue := UpdateNodeUpgradeJobStatus(oldValue, status)

	// after mark each node upgrade state, we also need to judge whether all edge node upgrade is completed
	// if all edge node is in completed state, we should set the total state to completed
	var completed int
	for _, v := range newValue.Status.Status {
		if v.State == v1alpha1.Completed {
			completed++
		}
	}
	if completed == len(newValue.Status.Status) {
		newValue.Status.State = v1alpha1.Completed
	} else {
		newValue.Status.State = v1alpha1.Upgrading
	}

	oldData, err := json.Marshal(oldValue)
	if err != nil {
		return fmt.Errorf("failed to marshal the old NodeUpgradeJob(%s): %v", oldValue.Name, err)
	}

	newData, err := json.Marshal(newValue)
	if err != nil {
		return fmt.Errorf("failed to marshal the new NodeUpgradeJob(%s): %v", newValue.Name, err)
	}

	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return fmt.Errorf("failed to create a merge patch: %v", err)
	}

	_, err = crdClient.OperationsV1alpha1().NodeUpgradeJobs().Patch(context.TODO(), newValue.Name, apimachineryType.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
	if err != nil {
		return fmt.Errorf("failed to patch update NodeUpgradeJob status: %v", err)
	}

	return nil
}
