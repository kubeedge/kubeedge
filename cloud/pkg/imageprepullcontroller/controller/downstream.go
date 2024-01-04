/*
Copyright 2023 The KubeEdge Authors.

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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	"github.com/kubeedge/kubeedge/cloud/pkg/common/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/imageprepullcontroller/manager"
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

	imagePrePullJobManager *manager.ImagePrePullJobManager
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	klog.Info("Start ImagePrePullJob Downstream Controller")
	go dc.syncImagePrePullJob()
	return nil
}

// syncImagePrePullJob is used to get events from informer
func (dc *DownstreamController) syncImagePrePullJob() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop sync ImagePrePullJob")
			return
		case e := <-dc.imagePrePullJobManager.Events():
			imagePrePull, ok := e.Object.(*v1alpha1.ImagePrePullJob)
			if !ok {
				klog.Warningf("object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				dc.imagePrePullJobAdded(imagePrePull)
			case watch.Deleted:
				dc.imagePrePullJobDeleted(imagePrePull)
			case watch.Modified:
				dc.imagePrePullJobUpdate(imagePrePull)
			default:
				klog.Warningf("ImagePrePullJob event type: %s unsupported", e.Type)
			}
		}
	}
}

// imagePrePullJobAdded is used to process addition of new ImagePrePullJob in apiserver
func (dc *DownstreamController) imagePrePullJobAdded(imagePrePull *v1alpha1.ImagePrePullJob) {
	klog.V(4).Infof("add ImagePrePullJob: %v", imagePrePull)
	// store in cache map
	dc.imagePrePullJobManager.ImagePrePullMap.Store(imagePrePull.Name, imagePrePull)

	// If imagePrePullJob is not initial state, we don't need to send message\
	if imagePrePull.Status.State != v1alpha1.PrePullInitialValue {
		klog.Errorf("The ImagePrePullJob %s is already running or completed, don't send message again", imagePrePull.Name)
		return
	}

	// get node list that need prepull images
	var nodesToPrePullImage []string
	if len(imagePrePull.Spec.ImagePrePullTemplate.NodeNames) != 0 {
		for _, node := range imagePrePull.Spec.ImagePrePullTemplate.NodeNames {
			nodeInfo, err := dc.informer.Core().V1().Nodes().Lister().Get(node)
			if err != nil {
				klog.Errorf("Failed to get node(%s) info: %v", node, err)
				continue
			}

			if validateNode(nodeInfo) {
				nodesToPrePullImage = append(nodesToPrePullImage, nodeInfo.Name)
			}
		}
	} else if imagePrePull.Spec.ImagePrePullTemplate.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(imagePrePull.Spec.ImagePrePullTemplate.LabelSelector)
		if err != nil {
			klog.Errorf("LabelSelector(%s) is not valid: %v", imagePrePull.Spec.ImagePrePullTemplate.LabelSelector, err)
			return
		}

		nodes, err := dc.informer.Core().V1().Nodes().Lister().List(selector)
		if err != nil {
			klog.Errorf("Failed to get nodes with label %s: %v", selector.String(), err)
			return
		}

		for _, node := range nodes {
			if validateNode(node) {
				nodesToPrePullImage = append(nodesToPrePullImage, node.Name)
			}
		}
	}

	// deduplicate: remove duplicate nodes to avoid repeating prepull images on the same node
	nodesToPrePullImage = util.RemoveDuplicateElement(nodesToPrePullImage)

	klog.Infof("Filtered finished, images will be prepulled on below nodes\n%v\n", nodesToPrePullImage)

	go func() {
		for _, node := range nodesToPrePullImage {
			dc.processPrePull(node, imagePrePull)
		}
	}()
}

// imagePrePullJobDeleted is used to process deleted ImagePrePullJob in apiServer
func (dc *DownstreamController) imagePrePullJobDeleted(imagePrePull *v1alpha1.ImagePrePullJob) {
	// delete drom cache map
	dc.imagePrePullJobManager.ImagePrePullMap.Delete(imagePrePull.Name)
}

// imagePrePullJobUpdate is used to process update of ImagePrePullJob in apiServer
// Now we don't allow update spec, so we only update the cache map in imagePrePullJobUpdate func.
func (dc *DownstreamController) imagePrePullJobUpdate(imagePrePull *v1alpha1.ImagePrePullJob) {
	_, ok := dc.imagePrePullJobManager.ImagePrePullMap.Load(imagePrePull.Name)
	// store in cache map
	dc.imagePrePullJobManager.ImagePrePullMap.Store(imagePrePull.Name, imagePrePull)
	if !ok {
		klog.Infof("ImagePrePull Job %s not exist, and store it into first", imagePrePull.Name)
		// If ImagePrePullJob not present in ImagePrePull map means it is not modified and added.
		dc.imagePrePullJobAdded(imagePrePull)
	}
}

// processPrePull process prepull job added and send it to edge nodes.
func (dc *DownstreamController) processPrePull(node string, imagePrePull *v1alpha1.ImagePrePullJob) {
	klog.V(4).Infof("begin to send imagePrePull message to %s", node)

	imagePrePullTemplateInfo := imagePrePull.Spec.ImagePrePullTemplate
	imagePrePullRequest := commontypes.ImagePrePullJobRequest{
		Images:     imagePrePullTemplateInfo.Images,
		NodeName:   node,
		Secret:     imagePrePullTemplateInfo.ImageSecret,
		RetryTimes: imagePrePullTemplateInfo.RetryTimes,
		CheckItems: imagePrePullTemplateInfo.CheckItems,
	}

	// handle timeout: could not receive image prepull msg feedback from edge node
	// send prepull timeout response message to upstream
	go dc.handleImagePrePullJobTimeoutOnEachNode(node, imagePrePull.Name, imagePrePullTemplateInfo.TimeoutSecondsOnEachNode)

	// send prepull msg to edge node
	msg := model.NewMessage("")
	resource := buildPrePullResource(imagePrePull.Name, node)
	msg.BuildRouter(modules.ImagePrePullControllerModuleName, modules.ImagePrePullControllerModuleGroup, resource, ImagePrePull).
		FillBody(imagePrePullRequest)

	err := dc.messageLayer.Send(*msg)
	if err != nil {
		klog.Errorf("Failed to send prepull message %s due to error %v", msg.GetID(), err)
		return
	}

	// update imagePrePullJob status to prepulling
	status := &v1alpha1.ImagePrePullStatus{
		NodeName: node,
		State:    v1alpha1.PrePulling,
	}
	err = patchImagePrePullStatus(dc.crdClient, imagePrePull, status)
	if err != nil {
		klog.Errorf("Failed to mark imagePrePullJob prepulling status: %v", err)
	}
}

// handleImagePrePullJobTimeoutOnEachNode is used to handle the situation that cloud don't receive prepull result
// from edge node within the timeout period.
// If so, the ImagePrePullJobState will update to timeout
func (dc *DownstreamController) handleImagePrePullJobTimeoutOnEachNode(node, jobName string, timeoutSecondsEachNode *uint32) {
	var timeout uint32 = 360
	if timeoutSecondsEachNode != nil && *timeoutSecondsEachNode != 0 {
		timeout = *timeoutSecondsEachNode
	}

	receiveFeedback := false

	_ = wait.Poll(10*time.Second, time.Duration(timeout)*time.Second, func() (bool, error) {
		cacheValue, ok := dc.imagePrePullJobManager.ImagePrePullMap.Load(jobName)
		if !ok {
			receiveFeedback = true
			klog.Errorf("ImagePrePullJob %s is not exist", jobName)
			return false, fmt.Errorf("imagePrePullJob %s is not exist", jobName)
		}
		imagePrePullValue := cacheValue.(*v1alpha1.ImagePrePullJob)
		for _, statusValue := range imagePrePullValue.Status.Status {
			if statusValue.NodeName == node && (statusValue.State == v1alpha1.PrePullSuccessful || statusValue.State == v1alpha1.PrePullFailed) {
				receiveFeedback = true
				return true, nil
			}
			break
		}
		return false, nil
	})

	if receiveFeedback {
		return
	}
	klog.Errorf("TIMEOUT to receive image prepull %s response from edge node %s", jobName, node)

	// construct timeout image prepull response and send it to upstream controller
	responseResource := buildPrePullResource(jobName, node)
	resp := commontypes.ImagePrePullJobResponse{
		NodeName: node,
		State:    v1alpha1.PrePullFailed,
		Reason:   "timeout to receive response from edge",
	}

	respMsg := model.NewMessage("").
		BuildRouter(modules.ImagePrePullControllerModuleName, modules.ImagePrePullControllerModuleGroup, responseResource, ImagePrePull).
		FillBody(resp)
	beehiveContext.Send(modules.ImagePrePullControllerModuleName, *respMsg)
}

// NewDownstreamController new downstream controller to process downstream imageprepull msg to edge nodes.
func NewDownstreamController(crdInformerFactory crdinformers.SharedInformerFactory) (*DownstreamController, error) {
	imagePrePullJobManager, err := manager.NewImagePrePullJobManager(crdInformerFactory.Operations().V1alpha1().ImagePrePullJobs().Informer())
	if err != nil {
		klog.Warningf("Create ImagePrePullJob manager failed with error: %s", err)
		return nil, err
	}

	dc := &DownstreamController{
		kubeClient:             client.GetKubeClient(),
		informer:               informers.GetInformersManager().GetKubeInformerFactory(),
		crdClient:              client.GetCRDClient(),
		imagePrePullJobManager: imagePrePullJobManager,
		messageLayer:           messagelayer.ImagePrePullControllerMessageLayer(),
	}
	return dc, nil
}
