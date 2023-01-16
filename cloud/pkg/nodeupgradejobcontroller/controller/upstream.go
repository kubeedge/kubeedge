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
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryType "k8s.io/apimachinery/pkg/types"
	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/nodeupgradejobcontroller/config"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
)

// UpstreamController subscribe messages from edge and sync to k8s api server
type UpstreamController struct {
	// downstream controller to update NodeUpgradeJob status in cache
	dc *DownstreamController

	kubeClient   kubernetes.Interface
	informer     k8sinformer.SharedInformerFactory
	crdClient    crdClientset.Interface
	messageLayer messagelayer.MessageLayer
	// message channel
	nodeUpgradeJobStatusChan chan model.Message
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	klog.Info("Start NodeUpgradeJob Upstream Controller")

	uc.nodeUpgradeJobStatusChan = make(chan model.Message, config.Config.Buffer.UpdateNodeUpgradeJobStatus)
	go uc.dispatchMessage()

	for i := 0; i < int(config.Config.Load.NodeUpgradeJobWorkers); i++ {
		go uc.updateNodeUpgradeJobStatus()
	}
	return nil
}

// Start UpstreamController
func (uc *UpstreamController) dispatchMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop dispatch NodeUpgradeJob upstream message")
			return
		default:
		}

		msg, err := uc.messageLayer.Receive()
		if err != nil {
			klog.Warningf("Receive message failed, %v", err)
			continue
		}

		klog.V(4).Infof("NodeUpgradeJob upstream controller receive msg %#v", msg)

		uc.nodeUpgradeJobStatusChan <- msg
	}
}

// updateNodeUpgradeJobStatus update NodeUpgradeJob status field
func (uc *UpstreamController) updateNodeUpgradeJobStatus() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop update NodeUpgradeJob status")
			return
		case msg := <-uc.nodeUpgradeJobStatusChan:
			klog.V(4).Infof("Message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

			// get nodeID and upgradeID from Upgrade msg:
			nodeID := getNodeName(msg.GetResource())
			upgradeID := getUpgradeID(msg.GetResource())

			oldValue, ok := uc.dc.nodeUpgradeJobManager.UpgradeMap.Load(upgradeID)
			if !ok {
				klog.Errorf("NodeUpgradeJob %s not exist", upgradeID)
				continue
			}

			upgrade, ok := oldValue.(*v1alpha1.NodeUpgradeJob)
			if !ok {
				klog.Errorf("NodeUpgradeJob info %T is not valid", oldValue)
				continue
			}

			data, err := msg.GetContentData()
			if err != nil {
				klog.Errorf("failed to get node upgrade content data: %v", err)
				continue
			}
			resp := &types.NodeUpgradeJobResponse{}
			err = json.Unmarshal(data, resp)
			if err != nil {
				klog.Errorf("Failed to unmarshal node upgrade response: %v", err)
				continue
			}

			status := &v1alpha1.UpgradeStatus{
				NodeName: nodeID,
				State:    v1alpha1.Completed,
				History: v1alpha1.History{
					HistoryID:   resp.HistoryID,
					FromVersion: resp.FromVersion,
					ToVersion:   resp.ToVersion,
					Result:      v1alpha1.UpgradeResult(resp.Status),
					Reason:      resp.Reason,
				},
			}
			err = patchNodeUpgradeJobStatus(uc.crdClient, upgrade, status)
			if err != nil {
				klog.Errorf("Failed to mark NodeUpgradeJob status to completed: %v", err)
			}

			// The below are to mark edge node schedulable
			// And to keep a successful record in node annotation only when upgrade is successful
			// like: nodeupgradejob.operations.kubeedge.io/history: "v1.9.0->v1.10.0->v1.11.1"
			nodeInfo, err := uc.informer.Core().V1().Nodes().Lister().Get(nodeID)
			if err != nil {
				klog.Errorf("Failed to get node info: %v", err)
				continue
			}

			// mark edge node schedulable
			// the effect is like running cmd: kubectl uncordon <node-to-drain>
			if nodeInfo.Labels != nil {
				if value, ok := nodeInfo.Labels[NodeUpgradeJobStatusKey]; ok {
					if value == NodeUpgradeJobStatusValue {
						nodeInfo.Spec.Unschedulable = false
						delete(nodeInfo.Labels, NodeUpgradeJobStatusKey)
					}
				}
			}
			// record upgrade logs in node annotation
			if v1alpha1.UpgradeResult(resp.Status) == v1alpha1.UpgradeSuccess {
				if nodeInfo.Annotations == nil {
					nodeInfo.Annotations = make(map[string]string)
				}
				nodeInfo.Annotations[NodeUpgradeHistoryKey] = mergeAnnotationUpgradeHistory(nodeInfo.Annotations[NodeUpgradeHistoryKey], resp.FromVersion, resp.ToVersion)
			}
			_, err = uc.kubeClient.CoreV1().Nodes().Update(context.Background(), nodeInfo, metav1.UpdateOptions{})
			if err != nil {
				// just log, and continue to process the next step
				klog.Errorf("Failed to mark node schedulable and add upgrade record: %v", err)
			}
		}
	}
}

// patchNodeUpgradeJobStatus call patch api to patch update NodeUpgradeJob status
func patchNodeUpgradeJobStatus(crdClient crdClientset.Interface, upgrade *v1alpha1.NodeUpgradeJob, status *v1alpha1.UpgradeStatus) error {
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

func getNodeName(resource string) string {
	// upgrade/${UpgradeID}/node/${NodeID}
	s := strings.Split(resource, "/")
	return s[3]
}
func getUpgradeID(resource string) string {
	// upgrade/${UpgradeID}/node/${NodeID}
	s := strings.Split(resource, "/")
	return s[1]
}

// NewUpstreamController create UpstreamController from config
func NewUpstreamController(dc *DownstreamController) (*UpstreamController, error) {
	uc := &UpstreamController{
		kubeClient:   keclient.GetKubeClient(),
		informer:     informers.GetInformersManager().GetKubeInformerFactory(),
		crdClient:    keclient.GetCRDClient(),
		messageLayer: messagelayer.NodeUpgradeJobControllerMessageLayer(),
		dc:           dc,
	}
	return uc, nil
}
