/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To create mini-kubelet for edge deployment scenario,
This file is derived from K8S Kubelet code with reduced set of methods
Changes done are
1. Package edged got some functions from "k8s.io/kubernetes/pkg/kubelet/kubelet.go"
and made some variant
*/

package edged

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/jsonpb"
	v1 "k8s.io/api/core/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
	"k8s.io/klog/v2"
	kubeletserver "k8s.io/kubernetes/cmd/kubelet/app"
	kubeletoptions "k8s.io/kubernetes/cmd/kubelet/app/options"
	"k8s.io/kubernetes/pkg/kubelet"
	"k8s.io/kubernetes/pkg/kubelet/config"
	"k8s.io/kubernetes/pkg/kubelet/nodestatus"
	kubelettypes "k8s.io/kubernetes/pkg/kubelet/types"
	csiplugin "k8s.io/kubernetes/pkg/volume/csi"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
	edgedconfig "github.com/kubeedge/kubeedge/edge/pkg/edged/config"
	kubebridge "github.com/kubeedge/kubeedge/edge/pkg/edged/kubeclientbridge"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	metaclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/version"
)

// edged is the main edged implementation.
type edged struct {
	enable      bool
	KuberServer *kubeletoptions.KubeletServer
	KubeletDeps *kubelet.Dependencies
	FeatureGate featuregate.FeatureGate
	context     context.Context
	nodeName    string
	namespace   string
}

var _ core.Module = (*edged)(nil)

// Register register edged
func Register(e *v1alpha1.Edged) {
	edgedconfig.InitConfigure(e)
	edged, err := newEdged(e.KubeletServer.EnableServer, e.HostnameOverride, e.RegisterNodeNamespace)
	if err != nil {
		klog.Errorf("init new edged error, %v", err)
		os.Exit(1)
	}
	core.Register(edged)
}

func (e *edged) Name() string {
	return modules.EdgedModuleName
}

func (e *edged) Group() string {
	return modules.EdgedGroup
}

//Enable indicates whether this module is enabled
func (e *edged) Enable() bool {
	return e.KuberServer.EnableServer
}

func (e *edged) Start() {
	klog.Info("Starting edged...")

	go func() {
		err := kubeletserver.Run(e.context, e.KuberServer, e.KubeletDeps, e.FeatureGate)
		if err != nil {
			klog.Errorf("Start edged failed, err: %v", err)
			os.Exit(1)
		}
	}()
	e.syncPod(e.KubeletDeps.PodConfig)
}

// newEdged creates new edged object and initialises it
func newEdged(enable bool, nodeName, namespace string) (*edged, error) {
	var ed *edged
	var err error
	if !enable {
		return &edged{
			enable:    enable,
			nodeName:  nodeName,
			namespace: namespace,
		}, nil
	}

	kubeletServer := edgedconfig.Config.KubeletServer
	nodestatus.KubeletVersion = fmt.Sprintf("%s-kubeedge-%s", constants.CurrentSupportK8sVersion, version.Get())
	// use kubeletServer to construct the default KubeletDeps
	kubeletDeps, err := kubeletserver.UnsecuredDependencies(&kubeletServer, utilfeature.DefaultFeatureGate)
	if err != nil {
		klog.ErrorS(err, "Failed to construct kubelet dependencies")
		return nil, fmt.Errorf("failed to construct kubelet dependencies")
	}
	MakeKubeClientBridge(kubeletDeps)

	// source of all configuration
	kubeletDeps.PodConfig = config.NewPodConfig(config.PodConfigNotificationIncremental, kubeletDeps.Recorder)

	ed = &edged{
		context:     context.Background(),
		KuberServer: &kubeletServer,
		KubeletDeps: kubeletDeps,
		FeatureGate: utilfeature.DefaultFeatureGate,
		nodeName:    nodeName,
		namespace:   namespace,
	}

	return ed, nil
}

func (e *edged) syncPod(podCfg *config.PodConfig) {
	time.Sleep(10 * time.Second)

	//when starting, send msg to metamanager once to get existing pods
	info := model.NewMessage("").BuildRouter(e.Name(), e.Group(), e.namespace+"/"+model.ResourceTypePod,
		model.QueryOperation)
	beehiveContext.Send(modules.MetaManagerModuleName, *info)
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Sync pod stop")
			return
		default:
		}
		result, err := beehiveContext.Receive(e.Name())
		if err != nil {
			klog.Errorf("failed to get pod: %v", err)
			continue
		}

		_, resType, resID, err := util.ParseResourceEdge(result.GetResource(), result.GetOperation())
		if err != nil {
			klog.Errorf("failed to parse the Resource: %v", err)
			continue
		}
		op := result.GetOperation()

		content, err := result.GetContentData()
		if err != nil {
			klog.Errorf("get message content data failed: %v", err)
			continue
		}

		switch resType {
		case model.ResourceTypePod:
			if op == model.ResponseOperation && resID == "" && result.GetSource() == modules.MetaManagerModuleName {
				err := e.handlePodListFromMetaManager(content, podCfg)
				if err != nil {
					klog.Errorf("handle podList failed: %v", err)
					continue
				}
				podCfg.SetInitPodReady(true)
			} else if op == model.ResponseOperation && resID == "" && result.GetSource() == metamanager.CloudControllerModel {
				err := e.handlePodListFromEdgeController(content, podCfg)
				if err != nil {
					klog.Errorf("handle podList failed: %v", err)
					continue
				}
				podCfg.SetInitPodReady(true)
			} else {
				err = e.handlePod(op, content, podCfg)
				if err != nil {
					klog.Errorf("handle pod failed: %v", err)
					continue
				}
			}
		case constants.CSIResourceTypeVolume:
			klog.Infof("volume operation type: %s", op)
			res, err := e.handleVolume(op, content)
			if err != nil {
				klog.Errorf("handle volume failed: %v", err)
			} else {
				resp := result.NewRespByMessage(&result, res)
				beehiveContext.SendResp(*resp)
			}
		default:
			klog.Errorf("resType is not pod or configmap or secret or volume: resType is %s", resType)
			continue
		}
	}
}

// MakeKubeClientBridge make kubeclient bridge to replace kubeclient with metaclient
func MakeKubeClientBridge(kubeletDeps *kubelet.Dependencies) {
	client := kubebridge.NewSimpleClientset(metaclient.New())

	kubeletDeps.KubeClient = client
	kubeletDeps.EventClient = nil
	kubeletDeps.HeartbeatClient = client
}

func (e *edged) handlePod(op string, content []byte, podCfg *config.PodConfig) (err error) {
	var pod v1.Pod
	err = json.Unmarshal(content, &pod)
	if err != nil {
		return err
	}

	var pods []*v1.Pod
	pods = append(pods, &pod)

	if filterPodByNodeName(&pod, e.nodeName) {
		switch op {
		case model.InsertOperation:
			adds := &kubelettypes.PodUpdate{Op: kubelettypes.ADD, Pods: pods, Source: kubelettypes.ApiserverSource}
			podCfg.EdgedCh <- *adds
		case model.UpdateOperation:
			updates := &kubelettypes.PodUpdate{Op: kubelettypes.UPDATE, Pods: pods, Source: kubelettypes.ApiserverSource}
			podCfg.EdgedCh <- *updates
		case model.DeleteOperation:
			deletes := &kubelettypes.PodUpdate{Op: kubelettypes.REMOVE, Pods: pods, Source: kubelettypes.ApiserverSource}
			podCfg.EdgedCh <- *deletes
		}
	}

	return nil
}

func (e *edged) handlePodListFromMetaManager(content []byte, podCfg *config.PodConfig) (err error) {
	var lists []string
	err = json.Unmarshal(content, &lists)
	if err != nil {
		return err
	}

	var pods []*v1.Pod
	var podsUpdate []*v1.Pod

	for _, list := range lists {
		var pod v1.Pod
		err = json.Unmarshal([]byte(list), &pod)
		if err != nil {
			return err
		}

		// if edge-core stop or panic when pod is deleting, pod need add into podDeletionQueue after edge-core restart.
		if filterPodByNodeName(&pod, e.nodeName) {
			if pod.DeletionTimestamp == nil {
				pods = append(pods, &pod)
			} else {
				podsUpdate = append(podsUpdate, &pod)
			}
		}
	}

	adds := &kubelettypes.PodUpdate{Op: kubelettypes.ADD, Pods: pods, Source: kubelettypes.ApiserverSource}
	podCfg.EdgedCh <- *adds

	updates := &kubelettypes.PodUpdate{Op: kubelettypes.UPDATE, Pods: podsUpdate, Source: kubelettypes.ApiserverSource}
	podCfg.EdgedCh <- *updates

	return nil
}

func (e *edged) handlePodListFromEdgeController(content []byte, podCfg *config.PodConfig) (err error) {
	var podLists []v1.Pod
	var pods []*v1.Pod
	if err := json.Unmarshal(content, &podLists); err != nil {
		return err
	}

	for _, pod := range podLists {
		if filterPodByNodeName(&pod, e.nodeName) {
			pods = append(pods, &pod)
			adds := &kubelettypes.PodUpdate{Op: kubelettypes.ADD, Pods: pods, Source: kubelettypes.ApiserverSource}
			podCfg.EdgedCh <- *adds
		}
	}

	return nil
}

func (e *edged) handleVolume(op string, content []byte) (interface{}, error) {
	switch op {
	case constants.CSIOperationTypeCreateVolume:
		return e.createVolume(content)
	case constants.CSIOperationTypeDeleteVolume:
		return e.deleteVolume(content)
	case constants.CSIOperationTypeControllerPublishVolume:
		return e.controllerPublishVolume(content)
	case constants.CSIOperationTypeControllerUnpublishVolume:
		return e.controllerUnpublishVolume(content)
	}
	return nil, nil
}

func (e *edged) createVolume(content []byte) (interface{}, error) {
	req := &csi.CreateVolumeRequest{}
	err := jsonpb.Unmarshal(bytes.NewReader(content), req)
	if err != nil {
		klog.Errorf("unmarshal create volume req error: %v", err)
		return nil, err
	}

	klog.V(4).Infof("start create volume: %s", req.Name)
	ctl := csiplugin.NewController()
	res, err := ctl.CreateVolume(req)
	if err != nil {
		klog.Errorf("create volume error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("end create volume: %s result: %v", req.Name, res)
	return res, nil
}

func (e *edged) deleteVolume(content []byte) (interface{}, error) {
	req := &csi.DeleteVolumeRequest{}
	err := jsonpb.Unmarshal(bytes.NewReader(content), req)
	if err != nil {
		klog.Errorf("unmarshal delete volume req error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("start delete volume: %s", req.VolumeId)
	ctl := csiplugin.NewController()
	res, err := ctl.DeleteVolume(req)
	if err != nil {
		klog.Errorf("delete volume error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("end delete volume: %s result: %v", req.VolumeId, res)
	return res, nil
}

func (e *edged) controllerPublishVolume(content []byte) (interface{}, error) {
	req := &csi.ControllerPublishVolumeRequest{}
	err := jsonpb.Unmarshal(bytes.NewReader(content), req)
	if err != nil {
		klog.Errorf("unmarshal controller publish volume req error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("start controller publish volume: %s", req.VolumeId)
	ctl := csiplugin.NewController()
	res, err := ctl.ControllerPublishVolume(req)
	if err != nil {
		klog.Errorf("controller publish volume error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("end controller publish volume:: %s result: %v", req.VolumeId, res)
	return res, nil
}

func (e *edged) controllerUnpublishVolume(content []byte) (interface{}, error) {
	req := &csi.ControllerUnpublishVolumeRequest{}
	err := jsonpb.Unmarshal(bytes.NewReader(content), req)
	if err != nil {
		klog.Errorf("unmarshal controller publish volume req error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("start controller unpublish volume: %s", req.VolumeId)
	ctl := csiplugin.NewController()
	res, err := ctl.ControllerUnpublishVolume(req)
	if err != nil {
		klog.Errorf("controller unpublish volume error: %v", err)
		return nil, err
	}
	klog.V(4).Infof("end controller unpublish volume:: %s result: %v", req.VolumeId, res)
	return res, nil
}

func filterPodByNodeName(pod *v1.Pod, nodeName string) bool {
	return pod.Spec.NodeName == nodeName
}
