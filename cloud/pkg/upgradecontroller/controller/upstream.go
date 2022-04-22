package controller

import (
	"context"
	"encoding/json"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/upgradecontroller/config"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/apis/upgrade/v1alpha2"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
)

// UpstreamController subscribe messages from edge and sync to k8s api server
type UpstreamController struct {
	kubeClient   kubernetes.Interface
	crdClient    crdClientset.Interface
	messageLayer messagelayer.MessageLayer
	// message channel
	upgradeStatusChan chan model.Message

	// downstream controller to update Upgrade status in cache
	dc *DownstreamController
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	klog.Info("Start upstream Upgrade Controller")

	uc.upgradeStatusChan = make(chan model.Message, config.Config.Buffer.UpdateUpgradeStatus)
	go uc.dispatchMessage()

	for i := 0; i < int(config.Config.Load.UpgradeWorkers); i++ {
		go uc.updateUpgradeStatus()
	}
	return nil
}

// Start UpstreamController
func (uc *UpstreamController) dispatchMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop dispatch upgrade upstream message")
			return
		default:
		}

		msg, err := uc.messageLayer.Receive()
		if err != nil {
			klog.Warningf("Receive message failed, %s", err)
			continue
		}

		klog.V(4).Infof("Upgrade upstream controller receive msg %#v", msg)

		uc.upgradeStatusChan <- msg
	}
}

// Start UpstreamController
func (uc *UpstreamController) updateUpgradeStatus() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop updateUpgradeStatus")
			return
		case msg := <-uc.upgradeStatusChan:
			klog.V(4).Infof("Message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

			// get nodeID and upgradeID from Upgrade msg:
			nodeID := getNodeName(msg.GetResource())
			upgradeID := getUpgradeID(msg.GetResource())

			oldValue, ok := uc.dc.upgradeManager.UpgradeMap.Load(upgradeID)
			if !ok {
				klog.Errorf("Upgrade %s not exist", upgradeID)
				continue
			}

			// mark node schedulable
			nodeInfo, err := uc.kubeClient.CoreV1().Nodes().Get(context.Background(), nodeID, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Failed to get node info: %v", err)
				continue
			}

			// mark edge node schedulable
			// the effect is like running cmd: kubectl uncordon <node-to-drain>
			if nodeInfo.Labels != nil && nodeInfo.Labels["upgrade"] == "upgrade" {
				nodeInfo.Spec.Unschedulable = false
				delete(nodeInfo.Labels, "upgrade")

				_, err = uc.kubeClient.CoreV1().Nodes().Update(context.Background(), nodeInfo, metav1.UpdateOptions{})
				if err != nil {
					// just log, and continue to process the next step
					klog.Errorf("Failed to mark node schedulable: %v", err)
				}
			}

			upgrade, ok := oldValue.(*v1alpha2.Upgrade)
			if !ok {
				klog.Errorf("upgrade info %T is not valid", oldValue)
				continue
			}

			data, err := msg.GetContentData()
			if err != nil {
				klog.Errorf("failed to get upgrade content data: %v", err)
				continue
			}
			resp := &types.UpgradeResponse{}
			err = json.Unmarshal(data, resp)
			if err != nil {
				continue
			}

			// for-range upgrade.Status to check whether node already exist
			exist := false
			status := v1alpha2.History{
				FromVersion: resp.FromVersion,
				ToVersion:   resp.ToVersion,
				Status:      v1alpha2.UpgradeOperationStatus(resp.Status),
				Reason:      resp.Reason,
			}

			for index := range upgrade.Status {
				if upgrade.Status[index].NodeName == nodeID {
					exist = true
					// we only keep the latest v1alpha2.MaxStatusHistory number Status history
					// if reach the v1alpha2.MaxStatusHistory, remove the oldest status, and insert the new one
					if len(upgrade.Status[index].History) >= v1alpha2.MaxStatusHistory {
						upgrade.Status[index].History = upgrade.Status[index].History[1:]
					}
					upgrade.Status[index].History = append(upgrade.Status[index].History, status)

					break
				}
			}
			if !exist {
				upgrade.Status = append(upgrade.Status, v1alpha2.UpgradeStatus{
					NodeName: nodeID,
					History:  []v1alpha2.History{status},
				})
			}

			// call k8s api to update status field
			_, err = uc.crdClient.UpgradeV1alpha2().Upgrades().UpdateStatus(context.Background(), upgrade, metav1.UpdateOptions{})
			if err != nil {
				klog.Errorf("failed to patch upgrade info: %v", err)
				continue
			}
		}
	}
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
		crdClient:    keclient.GetCRDClient(),
		messageLayer: messagelayer.UpgradeControllerMessageLayer(),
		dc:           dc,
	}
	return uc, nil
}
