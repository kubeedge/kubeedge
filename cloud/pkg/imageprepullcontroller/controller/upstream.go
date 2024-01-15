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
	"encoding/json"

	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/imageprepullcontroller/config"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
)

// UpstreamController subscribe messages from edge and sync to k8s api server
type UpstreamController struct {
	// downstream controller to update ImagePrePullJob status in cache
	dc *DownstreamController

	kubeClient   kubernetes.Interface
	informer     k8sinformer.SharedInformerFactory
	crdClient    crdClientset.Interface
	messageLayer messagelayer.MessageLayer
	// message channel
	imagePrePullJobStatusChan chan model.Message
}

// Start imageprepull UpstreamController
func (uc *UpstreamController) Start() error {
	klog.Info("Start ImagePrePullJob Upstream Controller")

	uc.imagePrePullJobStatusChan = make(chan model.Message, config.Config.Buffer.ImagePrePullJobStatus)
	go uc.dispatchMessage()

	for i := 0; i < int(config.Config.Load.ImagePrePullJobWorkers); i++ {
		go uc.updateImagePrePullJobStatus()
	}
	return nil
}

// updateImagePrePullJobStatus update imagePrePullJob status
func (uc *UpstreamController) updateImagePrePullJobStatus() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop update ImagePrePullJob status")
			return
		case msg := <-uc.imagePrePullJobStatusChan:
			klog.V(4).Infof("Message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
			nodeName, jobName, err := parsePrePullresource(msg.GetResource())
			if err != nil {
				klog.Errorf("message resource %s is not supported", msg.GetResource())
				continue
			}

			oldValue, ok := uc.dc.imagePrePullJobManager.ImagePrePullMap.Load(jobName)
			if !ok {
				klog.Errorf("ImagePrePullJob %s not exist", jobName)
				continue
			}

			imagePrePull, ok := oldValue.(*v1alpha1.ImagePrePullJob)
			if !ok {
				klog.Errorf("ImagePrePullJob info %T is not valid", oldValue)
				continue
			}

			data, err := msg.GetContentData()
			if err != nil {
				klog.Errorf("failed to get image prepull content from response msg, err: %v", err)
				continue
			}
			resp := &types.ImagePrePullJobResponse{}
			err = json.Unmarshal(data, resp)
			if err != nil {
				klog.Errorf("Failed to unmarshal image prepull response: %v", err)
				continue
			}

			status := &v1alpha1.ImagePrePullStatus{
				NodeName:    nodeName,
				State:       resp.State,
				Reason:      resp.Reason,
				ImageStatus: resp.ImageStatus,
			}
			err = patchImagePrePullStatus(uc.crdClient, imagePrePull, status)
			if err != nil {
				klog.Errorf("Failed to patch ImagePrePullJob status, err: %v", err)
			}
		}
	}
}

// dispatchMessage receive the message from edge and write into imagePrePullJobStatusChan
func (uc *UpstreamController) dispatchMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop dispatch ImagePrePullJob upstream message")
			return
		default:
		}

		msg, err := uc.messageLayer.Receive()
		if err != nil {
			klog.Warningf("Receive message failed, %v", err)
			continue
		}

		klog.V(4).Infof("ImagePrePullJob upstream controller receive msg %#v", msg)
		uc.imagePrePullJobStatusChan <- msg
	}
}

// NewUpstreamController create UpstreamController from config
func NewUpstreamController(dc *DownstreamController) (*UpstreamController, error) {
	uc := &UpstreamController{
		kubeClient:   keclient.GetKubeClient(),
		informer:     informers.GetInformersManager().GetKubeInformerFactory(),
		crdClient:    keclient.GetCRDClient(),
		messageLayer: messagelayer.ImagePrePullControllerMessageLayer(),
		dc:           dc,
	}
	return uc, nil
}
