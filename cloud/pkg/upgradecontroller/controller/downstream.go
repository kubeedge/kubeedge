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
	"reflect"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryType "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/upgradecontroller/manager"
	"github.com/kubeedge/kubeedge/common/constants"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/apis/upgrade/v1alpha2"
	crdinformers "github.com/kubeedge/kubeedge/pkg/client/informers/externalversions"
)

type DownstreamController struct {
	kubeClient   kubernetes.Interface
	messageLayer messagelayer.MessageLayer

	upgradeManager *manager.UpgradeManager
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	klog.Info("Start downstream Upgrade Controller")

	go dc.syncUpgrade()

	return nil
}

// syncUpgrade is used to get events from informer
func (dc *DownstreamController) syncUpgrade() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop sync Upgrade")
			return
		case e := <-dc.upgradeManager.Events():
			upgrade, ok := e.Object.(*v1alpha2.Upgrade)
			if !ok {
				klog.Warningf("object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				dc.upgradeAdded(upgrade)
			case watch.Deleted:
				dc.upgradeDeleted(upgrade)
			case watch.Modified:
				dc.upgradeUpdated(upgrade)
			default:
				klog.Warningf("upgrade event type: %s unsupported", e.Type)
			}
		}
	}
}

func buildUpgradeResource(upgradeID, nodeID string) string {
	resource := fmt.Sprintf("%s%s%s%s%s%s%s", "upgrade", constants.ResourceSep, upgradeID, constants.ResourceSep, "node", constants.ResourceSep, nodeID)
	return resource
}

// upgradeAdded is used to process addition of new Upgrade in apiserver
func (dc *DownstreamController) upgradeAdded(upgrade *v1alpha2.Upgrade) {
	klog.V(4).Infof("create upgrade CR: %v", upgrade)
	// store in cache map
	dc.upgradeManager.UpgradeMap.Store(upgrade.Name, upgrade)

	// get node list that need upgrading
	var nodeName []string
	if len(upgrade.Spec.NodeNames) != 0 {
		for _, node := range upgrade.Spec.NodeNames {
			nodeInfo, err := dc.kubeClient.CoreV1().Nodes().Get(context.Background(), node, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Upgrade failed: failed to get nodes: %v", err)
				continue
			}
			if filterVersion(nodeInfo.Status.NodeInfo.KubeletVersion, upgrade.Spec.Version) {
				klog.Warningf("Node(%s) version(%s) already on the expected version %s.", node, nodeInfo.Status.NodeInfo.KubeletVersion, upgrade.Spec.Version)
				continue
			}

			nodeName = append(nodeName, node)
		}
	} else if upgrade.Spec.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(upgrade.Spec.LabelSelector)
		if err != nil {
			klog.Errorf("Upgrade failed: labelSelector is not valid: %v", err)
			return
		}
		nodes, err := dc.kubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			klog.Errorf("Upgrade failed: failed to get nodes with label %s: %v", selector.String(), err)
			return
		}

		for _, node := range nodes.Items {
			if filterVersion(node.Status.NodeInfo.KubeletVersion, upgrade.Spec.Version) {
				klog.Warningf("Node(%s) version(%s) already on the expected version %s.", node, node.Status.NodeInfo.KubeletVersion, upgrade.Spec.Version)
				continue
			}

			nodeName = append(nodeName, node.Name)
		}
	} else {
		klog.Errorf("Upgrade prohibit, no valid nodes are specified.")
		return
	}

	// deduplicate: remove duplicate nodes to avoid repeating upgrade to the same node
	nodeName = RemoveDuplicateElement(nodeName)

	klog.Infof("Filtered finished, the below nodes are to upgrade\n%v", nodeName)

	for _, node := range nodeName {
		// send upgrade msg to every edge node
		msg := model.NewMessage("")

		resource := buildUpgradeResource(upgrade.Name, node)
		upgradeReq := commontypes.UpgradeRequest{
			UpgradeID:           upgrade.Name,
			UpgradeCmd:          upgrade.Spec.UpgradeCmd,
			UpgradeInstallerCmd: upgrade.Spec.UpgradeInstaller,
			Version:             upgrade.Spec.Version,
		}

		msg.BuildRouter(modules.UpgradeControllerModuleName, modules.UpgradeControllerModuleGroup, resource, "upgrade").
			FillBody(upgradeReq)

		err := dc.messageLayer.Send(*msg)
		if err != nil {
			klog.Errorf("Failed to send upgrade message %v due to error %v", msg, err)
			continue
		}

		// mark edge node unschedulable
		// the effect is like running cmd: kubectl drain <node-to-drain> --ignore-daemonsets
		unscheduleNode := v1.Node{}
		unscheduleNode.Spec.Unschedulable = true
		// add a upgrade label
		unscheduleNode.Labels = map[string]string{"upgrade": "upgrade"}
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

		// time out
	}
}

// upgradeDeleted is used to process deleted Upgrade in apiserver
func (dc *DownstreamController) upgradeDeleted(upgrade *v1alpha2.Upgrade) {
	// just need to delete from cache map
	dc.upgradeManager.UpgradeMap.Delete(upgrade.Name)
}

// upgradeAdded is used to process update of new Upgrade in apiserver
func (dc *DownstreamController) upgradeUpdated(upgrade *v1alpha2.Upgrade) {
	oldValue, ok := dc.upgradeManager.UpgradeMap.Load(upgrade.Name)
	// store in cache map
	dc.upgradeManager.UpgradeMap.Store(upgrade.Name, upgrade)
	if !ok {
		klog.Errorf("Upgrade %s not exist, cannot be updated.", upgrade.Name)
		return
	}

	old := oldValue.(*v1alpha2.Upgrade)
	if !isUpgradeUpdated(upgrade, old) {
		klog.Infof("Upgrade %s no need to update", upgrade.Name)
		return
	}

	dc.upgradeAdded(upgrade)
}

// isUpgradeUpdated checks Upgrade is updated or not
func isUpgradeUpdated(new *v1alpha2.Upgrade, old *v1alpha2.Upgrade) bool {
	// we only care spec fields
	// return true if Spec changed, else false
	return !reflect.DeepEqual(old.Spec, new.Spec)
}

func NewDownstreamController(crdInformerFactory crdinformers.SharedInformerFactory) (*DownstreamController, error) {
	upgradeManager, err := manager.NewUpgradeManager(crdInformerFactory.Upgrade().V1alpha2().Upgrades().Informer())
	if err != nil {
		klog.Warningf("Create Upgrade manager failed with error: %s", err)
		return nil, err
	}

	dc := &DownstreamController{
		kubeClient:     client.GetKubeClient(),
		upgradeManager: upgradeManager,
		messageLayer:   messagelayer.UpgradeControllerMessageLayer(),
	}
	return dc, nil
}
