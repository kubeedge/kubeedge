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
	"net/http"
	"os"
	"reflect"
	"strconv"
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
	kubeletconfig "k8s.io/kubernetes/pkg/kubelet/apis/config"
	"k8s.io/kubernetes/pkg/kubelet/config"
	"k8s.io/kubernetes/pkg/kubelet/nodestatus"
	kubelettypes "k8s.io/kubernetes/pkg/kubelet/types"
	csiplugin "k8s.io/kubernetes/pkg/volume/csi"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
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
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	kefeatures "github.com/kubeedge/kubeedge/pkg/features"
	"github.com/kubeedge/kubeedge/pkg/version"
)

// GetKubeletDeps returns a Dependencies suitable for lite kubelet being run.
type GetKubeletDeps func(
	s *kubeletoptions.KubeletServer,
	featureGate featuregate.FeatureGate) (*kubelet.Dependencies, error)

// RunLiteKubelet runs the specified lite kubelet with the given Dependencies.
type RunLiteKubelet func(
	ctx context.Context,
	s *kubeletoptions.KubeletServer,
	kubeDeps *kubelet.Dependencies,
	featureGate featuregate.FeatureGate) error

// DefaultKubeletDeps will only be changed when EdgeMark is enabled
var DefaultKubeletDeps GetKubeletDeps = kubeletserver.UnsecuredDependencies

// DefaultRunLiteKubelet will only be changed when EdgeMark is enabled
var DefaultRunLiteKubelet RunLiteKubelet = kubeletserver.Run

// edged is the main edged implementation.
type edged struct {
	enable        bool
	KubeletServer *kubeletoptions.KubeletServer
	KubeletDeps   *kubelet.Dependencies
	FeatureGate   featuregate.FeatureGate
	context       context.Context
	nodeName      string
	namespace     string
}

var _ core.Module = (*edged)(nil)

// Register register edged
func Register(e *v1alpha2.Edged) {
	edgedconfig.InitConfigure(e)
	edged, err := newEdged(e.Enable, e.HostnameOverride, e.RegisterNodeNamespace)
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

// Enable indicates whether this module is enabled
func (e *edged) Enable() bool {
	return edgedconfig.Config.Enable
}

func (e *edged) Start() {
	klog.Info("Starting edged...")

	// FIXME: cleanup this code when the static pod mqtt broker no longer needs to be compatible
	// edged saves the data of mqtt container in sqlite3 and starts it. This is a temporary workaround and will be modified in v1.15.
	withMqtt, err := strconv.ParseBool(os.Getenv(constants.DeployMqttContainerEnv))
	if err == nil && withMqtt {
		if err := dao.SaveMQTTMeta(e.nodeName); err != nil {
			klog.ErrorS(err, "Start mqtt container failed")
		}
	} else {
		// Delete a not exists key does not return an error
		if err := dao.DeleteMetaByKey(fmt.Sprintf("default/pod/%s", constants.DefaultMosquittoContainerName)); err != nil {
			klog.ErrorS(err, "delete mqtt container failed")
		}
	}

	kubeletErrChan := make(chan error, 1)
	go func() {
		err := DefaultRunLiteKubelet(e.context, e.KubeletServer, e.KubeletDeps, e.FeatureGate)
		if err != nil {
			if !kefeatures.DefaultFeatureGate.Enabled(kefeatures.ModuleRestart) {
				klog.Errorf("Start edged failed, err: %v", err)
				os.Exit(1)
			}
			kubeletErrChan <- err
		}
	}()

	kubeletReadyChan := make(chan struct{}, 1)
	go kubeletHealthCheck(e.KubeletServer.ReadOnlyPort, kubeletReadyChan)

	select {
	case <-beehiveContext.Done():
		klog.Warning("Stop sync pod")
		return
	case err := <-kubeletErrChan:
		klog.Errorf("Failed to start edged, err: %v", err)
		return
	case <-kubeletReadyChan:
		klog.Info("Start sync pod")
	}

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

	// initial kubelet config and flag
	var kubeletConfig kubeletconfig.KubeletConfiguration
	var kubeletFlags kubeletoptions.KubeletFlags
	err = edgedconfig.ConvertEdgedKubeletConfigurationToConfigKubeletConfiguration(edgedconfig.Config.TailoredKubeletConfig, &kubeletConfig, nil)
	if err != nil {
		klog.ErrorS(err, "Failed to convert kubelet config")
		return nil, fmt.Errorf("failed to construct kubelet configuration")
	}
	edgedconfig.ConvertConfigEdgedFlagToConfigKubeletFlag(&edgedconfig.Config.TailoredKubeletFlag, &kubeletFlags)

	// set feature gates from initial flags-based config
	if err := utilfeature.DefaultMutableFeatureGate.SetFromMap(kubeletConfig.FeatureGates); err != nil {
		return nil, fmt.Errorf("failed to set feature gates from initial flags-based config: %w", err)
	}

	// construct a KubeletServer from kubeletFlags and kubeletConfig
	kubeletServer := kubeletoptions.KubeletServer{
		KubeletFlags:         kubeletFlags,
		KubeletConfiguration: kubeletConfig,
	}

	// make directory for static pod
	if kubeletConfig.StaticPodPath != "" {
		if err := os.MkdirAll(kubeletConfig.StaticPodPath, os.ModePerm); err != nil {
			return nil, fmt.Errorf("create %s static pod path failed: %v", kubeletConfig.StaticPodPath, err)
		}
	} else {
		klog.ErrorS(err, "static pod path is nil!")
	}

	// set edged version
	nodestatus.KubeletVersion = fmt.Sprintf("%s-kubeedge-%s", constants.CurrentSupportK8sVersion, version.Get())

	// use kubeletServer to construct the default KubeletDeps
	kubeletDeps, err := DefaultKubeletDeps(&kubeletServer, utilfeature.DefaultFeatureGate)
	if err != nil {
		klog.ErrorS(err, "Failed to construct kubelet dependencies")
		return nil, fmt.Errorf("failed to construct kubelet dependencies")
	}
	MakeKubeClientBridge(kubeletDeps)

	// source of all configuration
	kubeletDeps.PodConfig = config.NewPodConfig(config.PodConfigNotificationIncremental, kubeletDeps.Recorder, kubeletDeps.PodStartupLatencyTracker)

	ed = &edged{
		enable:        true,
		context:       context.Background(),
		KubeletServer: &kubeletServer,
		KubeletDeps:   kubeletDeps,
		FeatureGate:   utilfeature.DefaultFeatureGate,
		nodeName:      nodeName,
		namespace:     namespace,
	}

	return ed, nil
}

func (e *edged) syncPod(podCfg *config.PodConfig) {
	//when starting, send msg to metamanager once to get existing pods
	info := model.NewMessage("").BuildRouter(e.Name(), e.Group(), e.namespace+"/"+model.ResourceTypePod,
		model.QueryOperation)
	beehiveContext.Send(modules.MetaManagerModuleName, *info)
	// rawUpdateChan receives the update events from metamanager or edgecontroller
	rawUpdateChan := podCfg.Channel(beehiveContext.GetContext(), kubelettypes.ApiserverSource)

	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop sync pod")
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
				err := e.handlePodListFromMetaManager(content, rawUpdateChan)
				if err != nil {
					klog.Errorf("handle podList failed: %v", err)
					continue
				}
				podCfg.SetInitPodReady(true)
			} else if op == model.ResponseOperation && resID == "" && result.GetSource() == metamanager.CloudControllerModel {
				err := e.handlePodListFromEdgeController(content, rawUpdateChan)
				if err != nil {
					klog.Errorf("handle podList failed: %v", err)
					continue
				}
				podCfg.SetInitPodReady(true)
			} else {
				err = e.handlePod(op, content, rawUpdateChan)
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
	kubeletDeps.HeartbeatClient = client
	if edgedconfig.Config.ReportEvent {
		kubeletDeps.EventClient = client.CoreV1()
	} else {
		kubeletDeps.EventClient = nil
	}
}

func (e *edged) handlePod(op string, content []byte, updatesChan chan<- interface{}) (err error) {
	var pod v1.Pod
	err = json.Unmarshal(content, &pod)
	if err != nil {
		return err
	}

	// When the edge node is offline and the pod in the node is deleted forcefully,
	// and then we make the node online, We do not have the pod full information
	// because the pod is deleted from the kube apiServer, then the syncController
	// will send a message with the pod name, namespace and UID, so we can not filter
	// pod according to the node name. So in this scenario, we query metadata from edge
	// database and use func handlePodListFromMetaManager to sync with Kubelet.
	if op == model.DeleteOperation && reflect.DeepEqual(pod.Spec, v1.PodSpec{}) {
		info := model.NewMessage("").BuildRouter(e.Name(), e.Group(), e.namespace+"/"+model.ResourceTypePod,
			model.QueryOperation)
		beehiveContext.Send(modules.MetaManagerModuleName, *info)
		return nil
	}

	var pods []*v1.Pod
	pods = append(pods, &pod)

	if filterPodByNodeName(&pod, e.nodeName) {
		var podOp kubelettypes.PodOperation
		switch op {
		case model.InsertOperation, model.UpdateOperation:
			klog.V(4).InfoS("Receive message of add/update pods", "operation", op, "pods", klog.KObjSlice(pods))
			podOp = kubelettypes.UPDATE
		case model.DeleteOperation:
			klog.V(4).InfoS("Receive message of deleting pods", "pods", klog.KObjSlice(pods))
			podOp = kubelettypes.REMOVE
		}
		updates := &kubelettypes.PodUpdate{Op: podOp, Pods: pods, Source: kubelettypes.ApiserverSource}
		updatesChan <- *updates
	}

	return nil
}

func (e *edged) handlePodListFromMetaManager(content []byte, updatesChan chan<- interface{}) (err error) {
	var lists []string
	err = json.Unmarshal(content, &lists)
	if err != nil {
		return err
	}

	var pods []*v1.Pod
	for _, list := range lists {
		var pod v1.Pod
		err = json.Unmarshal([]byte(list), &pod)
		if err != nil {
			return err
		}

		if filterPodByNodeName(&pod, e.nodeName) {
			pods = append(pods, &pod)
		}
	}

	updates := &kubelettypes.PodUpdate{Op: kubelettypes.SET, Pods: pods, Source: kubelettypes.ApiserverSource}
	updatesChan <- *updates

	return nil
}

func (e *edged) handlePodListFromEdgeController(content []byte, updatesChan chan<- interface{}) (err error) {
	var podLists []v1.Pod
	var pods []*v1.Pod
	if err := json.Unmarshal(content, &podLists); err != nil {
		return err
	}

	for _, pod := range podLists {
		if filterPodByNodeName(&pod, e.nodeName) {
			pods = append(pods, &pod)
		}
	}
	updates := &kubelettypes.PodUpdate{Op: kubelettypes.SET, Pods: pods, Source: kubelettypes.ApiserverSource}
	updatesChan <- *updates

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

func kubeletHealthCheck(port int32, kubeletReadyChan chan struct{}) {
	url := fmt.Sprintf("http://localhost:%d/healthz/syncloop", port)
	for {
		resp, err := http.Get(url)
		if err != nil {
			klog.Warningf("failed to get kubelet healthz syncloop, err: %v", err)
			time.Sleep(50 * time.Millisecond)
			continue
		}

		statusCode := resp.StatusCode
		err = resp.Body.Close()
		if err != nil {
			klog.Errorf("failed to close response's body with err:%v", err)
		}

		if statusCode != http.StatusOK {
			klog.Warningf("internal error and status code: %d", resp.StatusCode)
		} else {
			kubeletReadyChan <- struct{}{}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}
