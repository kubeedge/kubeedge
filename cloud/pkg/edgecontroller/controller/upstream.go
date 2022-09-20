/*
Copyright 2019 The KubeEdge Authors.
Copyright 2014 The Kubernetes Authors.

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
KubeEdge Authors: To manage node/pod status for edge deployment scenarios,
we grab some functions from `kubelet/status/status_manager.go and do some modifications, they are
1. updatePodStatus
2. updateNodeStatus
3. normalizePodStatus
4. isPodNotRunning
*/

package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"sort"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apimachineryType "k8s.io/apimachinery/pkg/types"
	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	coordinationlisters "k8s.io/client-go/listers/coordination/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/controller"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/types"
	routerrule "github.com/kubeedge/kubeedge/cloud/pkg/router/rule"
	common "github.com/kubeedge/kubeedge/common/constants"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	rulesv1 "github.com/kubeedge/kubeedge/pkg/apis/rules/v1"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// SortedContainerStatuses define A type to help sort container statuses based on container names.
type SortedContainerStatuses []v1.ContainerStatus

func (s SortedContainerStatuses) Len() int      { return len(s) }
func (s SortedContainerStatuses) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s SortedContainerStatuses) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// SortInitContainerStatuses ensures that statuses are in the order that their
// init container appears in the pod spec
func SortInitContainerStatuses(p *v1.Pod, statuses []v1.ContainerStatus) {
	containers := p.Spec.InitContainers
	current := 0
	for _, container := range containers {
		for j := current; j < len(statuses); j++ {
			if container.Name == statuses[j].Name {
				statuses[current], statuses[j] = statuses[j], statuses[current]
				current++
				break
			}
		}
	}
}

// ObjectResp is the object that api-server response
type ObjectResp struct {
	Object metaV1.Object
	Err    error
}

// UpstreamController subscribe messages from edge and sync to k8s api server
type UpstreamController struct {
	kubeClient   kubernetes.Interface
	messageLayer messagelayer.MessageLayer
	crdClient    crdClientset.Interface

	config v1alpha1.EdgeController

	// message channel
	nodeStatusChan            chan model.Message
	podStatusChan             chan model.Message
	secretChan                chan model.Message
	serviceAccountTokenChan   chan model.Message
	configMapChan             chan model.Message
	serviceChan               chan model.Message
	endpointsChan             chan model.Message
	persistentVolumeChan      chan model.Message
	persistentVolumeClaimChan chan model.Message
	volumeAttachmentChan      chan model.Message
	queryNodeChan             chan model.Message
	createNodeChan            chan model.Message
	patchNodeChan             chan model.Message
	updateNodeChan            chan model.Message
	patchPodChan              chan model.Message
	podDeleteChan             chan model.Message
	ruleStatusChan            chan model.Message
	createLeaseChan           chan model.Message
	queryLeaseChan            chan model.Message

	// lister
	podLister       corelisters.PodLister
	configMapLister corelisters.ConfigMapLister
	secretLister    corelisters.SecretLister
	serviceLister   corelisters.ServiceLister
	endpointLister  corelisters.EndpointsLister
	nodeLister      corelisters.NodeLister
	leaseLister     coordinationlisters.LeaseLister
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	klog.Info("start upstream controller")

	go uc.dispatchMessage()

	for i := 0; i < int(uc.config.Load.UpdateNodeStatusWorkers); i++ {
		go uc.updateNodeStatus()
	}
	for i := 0; i < int(uc.config.Load.UpdatePodStatusWorkers); i++ {
		go uc.updatePodStatus()
	}
	for i := 0; i < int(uc.config.Load.QueryConfigMapWorkers); i++ {
		go uc.queryConfigMap()
	}
	for i := 0; i < int(uc.config.Load.QuerySecretWorkers); i++ {
		go uc.querySecret()
	}
	for i := 0; i < int(uc.config.Load.ServiceAccountTokenWorkers); i++ {
		go uc.processServiceAccountToken()
	}
	for i := 0; i < int(uc.config.Load.QueryPersistentVolumeWorkers); i++ {
		go uc.queryPersistentVolume()
	}
	for i := 0; i < int(uc.config.Load.QueryPersistentVolumeClaimWorkers); i++ {
		go uc.queryPersistentVolumeClaim()
	}
	for i := 0; i < int(uc.config.Load.QueryVolumeAttachmentWorkers); i++ {
		go uc.queryVolumeAttachment()
	}
	for i := 0; i < int(uc.config.Load.CreateNodeWorkers); i++ {
		go uc.registerNode()
	}
	for i := 0; i < int(uc.config.Load.PatchNodeWorkers); i++ {
		go uc.patchNode()
	}
	for i := 0; i < int(uc.config.Load.QueryNodeWorkers); i++ {
		go uc.queryNode()
	}
	for i := 0; i < int(uc.config.Load.UpdateNodeWorkers); i++ {
		go uc.updateNode()
	}
	for i := 0; i < int(uc.config.Load.PatchPodWorkers); i++ {
		go uc.patchPod()
	}
	for i := 0; i < int(uc.config.Load.DeletePodWorkers); i++ {
		go uc.deletePod()
	}
	for i := 0; i < int(uc.config.Load.CreateLeaseWorkers); i++ {
		go uc.createOrUpdateLease()
	}
	for i := 0; i < int(uc.config.Load.QueryLeaseWorkers); i++ {
		go uc.queryLease()
	}
	for i := 0; i < int(uc.config.Load.UpdateRuleStatusWorkers); i++ {
		go uc.updateRuleStatus()
	}
	return nil
}

func (uc *UpstreamController) dispatchMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop dispatchMessage")
			return
		default:
		}
		msg, err := uc.messageLayer.Receive()
		if err != nil {
			klog.Warningf("receive message failed, %s", err)
			continue
		}

		klog.V(5).Infof("dispatch message ID: %s", msg.GetID())
		klog.V(5).Infof("dispatch message content: %+v", msg)

		resourceType, err := messagelayer.GetResourceType(msg)
		if err != nil {
			klog.Warningf("parse message: %s resource type with error, message resource: %s, err: %v", msg.GetID(), msg.GetResource(), err)
			continue
		}

		klog.V(5).Infof("message: %s, operation type is: %s", msg.GetID(), msg.GetOperation())

		switch resourceType {
		case model.ResourceTypeNodeStatus:
			uc.nodeStatusChan <- msg
		case model.ResourceTypePodStatus:
			uc.podStatusChan <- msg
		case model.ResourceTypeConfigmap:
			uc.configMapChan <- msg
		case model.ResourceTypeSecret:
			uc.secretChan <- msg
		case model.ResourceTypeServiceAccountToken:
			uc.serviceAccountTokenChan <- msg
		case common.ResourceTypePersistentVolume:
			uc.persistentVolumeChan <- msg
		case common.ResourceTypePersistentVolumeClaim:
			uc.persistentVolumeClaimChan <- msg
		case common.ResourceTypeVolumeAttachment:
			uc.volumeAttachmentChan <- msg
		case model.ResourceTypeNode:
			switch msg.GetOperation() {
			case model.InsertOperation:
				uc.createNodeChan <- msg
			case model.QueryOperation:
				uc.queryNodeChan <- msg
			case model.UpdateOperation:
				uc.updateNodeChan <- msg
			default:
				klog.Errorf("message: %s, operation type: %s unsupported", msg.GetID(), msg.GetOperation())
			}
		case model.ResourceTypeNodePatch:
			uc.patchNodeChan <- msg
		case model.ResourceTypePodPatch:
			uc.patchPodChan <- msg
		case model.ResourceTypePod:
			if msg.GetOperation() == model.DeleteOperation {
				uc.podDeleteChan <- msg
			} else {
				klog.Errorf("message: %s, operation type: %s unsupported", msg.GetID(), msg.GetOperation())
			}
		case model.ResourceTypeRuleStatus:
			uc.ruleStatusChan <- msg
		case model.ResourceTypeLease:
			switch msg.GetOperation() {
			case model.InsertOperation, model.UpdateOperation:
				uc.createLeaseChan <- msg
			case model.QueryOperation:
				uc.queryLeaseChan <- msg
			}
		default:
			klog.Errorf("message: %s, resource type: %s unsupported", msg.GetID(), resourceType)
		}
	}
}

func (uc *UpstreamController) updateRuleStatus() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop updateRuleStatus")
			return
		case msg := <-uc.ruleStatusChan:
			klog.V(5).Infof("message %s, operation is : %s , and resource is %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
			namespace, err := messagelayer.GetNamespace(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get namespace failed with error: %s", msg.GetID(), err)
				continue
			}
			ruleID, err := messagelayer.GetResourceName(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get resource name failed with error: %s", msg.GetID(), err)
				continue
			}
			var rule *rulesv1.Rule
			rule, err = uc.crdClient.RulesV1().Rules(namespace).Get(context.Background(), ruleID, metaV1.GetOptions{})
			if err != nil {
				klog.Warningf("message: %s process failure, get rule with error: %s, namespaces: %s name: %s", msg.GetID(), err, namespace, ruleID)
				continue
			}
			content, ok := msg.Content.(routerrule.ExecResult)
			if !ok {
				klog.Warningf("message: %s process failure, get rule content with error: %s, namespaces: %s name: %s", msg.GetID(), err, namespace, ruleID)
				continue
			}
			if content.Status == "SUCCESS" {
				rule.Status.SuccessMessages++
			}
			if content.Status == "FAIL" {
				rule.Status.FailMessages++
				errSlice := make([]string, 0)
				rule.Status.Errors = append(errSlice, content.Error.Detail)
			}
			newStatus := &rulesv1.RuleStatus{
				SuccessMessages: rule.Status.SuccessMessages,
				FailMessages:    rule.Status.FailMessages,
				Errors:          rule.Status.Errors,
			}
			body, err := json.Marshal(newStatus)
			if err != nil {
				klog.Warningf("message: %s process failure, content marshal err: %s", msg.GetID(), err)
				continue
			}
			_, err = uc.crdClient.RulesV1().Rules(namespace).Patch(context.Background(), ruleID, controller.MergePatchType, body, metaV1.PatchOptions{})
			if err != nil {
				klog.Warningf("message: %s process failure, update ruleStatus failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, namespace, ruleID)
			} else {
				klog.Infof("UpdateRulestatus successfully!")
			}
		}
	}
}

func (uc *UpstreamController) podStatusResponse(msg model.Message) {
	resMsg := model.NewMessage(msg.GetID()).
		FillBody(common.MessageSuccessfulContent).
		BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, msg.Router.Resource, model.ResponseOperation)

	if err := uc.messageLayer.Send(*resMsg); err != nil {
		klog.Warningf("Send message failed: %s, operation: %s, resource: %s", err, resMsg.GetOperation(), resMsg.GetResource())
	} else {
		klog.V(4).Infof("Send message successfully, operation: %s, resource: %s", resMsg.GetOperation(), resMsg.GetResource())
	}
}

func (uc *UpstreamController) updatePodStatus() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop updatePodStatus")
			return
		case msg := <-uc.podStatusChan:
			klog.V(5).Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

			namespace, podStatuses := uc.unmarshalPodStatusMessage(msg)
			switch msg.GetOperation() {
			case model.UpdateOperation:
				for _, podStatus := range podStatuses {
					getPod, err := uc.kubeClient.CoreV1().Pods(namespace).Get(context.Background(), podStatus.Name, metaV1.GetOptions{})
					if errors.IsNotFound(err) {
						klog.Warningf("message: %s, pod not found, namespace: %s, name: %s", msg.GetID(), namespace, podStatus.Name)

						// Send request to delete this pod on edge side
						delMsg := model.NewMessage("")
						nodeID, err := messagelayer.GetNodeID(msg)
						if err != nil {
							klog.Warningf("Get node ID failed with error: %s", err)
							continue
						}
						resource, err := messagelayer.BuildResource(nodeID, namespace, model.ResourceTypePod, podStatus.Name)
						if err != nil {
							klog.Warningf("Built message resource failed with error: %s", err)
							continue
						}
						pod := &v1.Pod{}
						pod.Namespace, pod.Name = namespace, podStatus.Name
						delMsg.Content = pod
						delMsg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
						if err := uc.messageLayer.Send(*delMsg); err != nil {
							klog.Warningf("Send message failed with error: %s, operation: %s, resource: %s", err, delMsg.GetOperation(), delMsg.GetResource())
						} else {
							klog.V(4).Infof("Send message successfully, operation: %s, resource: %s", delMsg.GetOperation(), delMsg.GetResource())
						}

						continue
					}
					if err != nil {
						klog.Warningf("message: %s, pod is nil, namespace: %s, name: %s, error: %s", msg.GetID(), namespace, podStatus.Name, err)
						continue
					}
					status := podStatus.Status
					oldStatus := getPod.Status
					// Set ReadyCondition.LastTransitionTime
					if readyCondition := uc.getPodCondition(&status, v1.PodReady); readyCondition != nil {
						// Need to set LastTransitionTime.
						lastTransitionTime := metaV1.Now()
						oldReadyCondition := uc.getPodCondition(&oldStatus, v1.PodReady)
						if oldReadyCondition != nil && readyCondition.Status == oldReadyCondition.Status {
							lastTransitionTime = oldReadyCondition.LastTransitionTime
						}
						readyCondition.LastTransitionTime = lastTransitionTime
					}

					// Set InitializedCondition.LastTransitionTime.
					if initCondition := uc.getPodCondition(&status, v1.PodInitialized); initCondition != nil {
						// Need to set LastTransitionTime.
						lastTransitionTime := metaV1.Now()
						oldInitCondition := uc.getPodCondition(&oldStatus, v1.PodInitialized)
						if oldInitCondition != nil && initCondition.Status == oldInitCondition.Status {
							lastTransitionTime = oldInitCondition.LastTransitionTime
						}
						initCondition.LastTransitionTime = lastTransitionTime
					}

					// ensure that the start time does not change across updates.
					if oldStatus.StartTime != nil && !oldStatus.StartTime.IsZero() {
						status.StartTime = oldStatus.StartTime
					} else if status.StartTime.IsZero() {
						// if the status has no start time, we need to set an initial time
						now := metaV1.Now()
						status.StartTime = &now
					}

					uc.normalizePodStatus(getPod, &status)
					getPod.Status = status

					if updatedPod, err := uc.kubeClient.CoreV1().Pods(getPod.Namespace).UpdateStatus(context.Background(), getPod, metaV1.UpdateOptions{}); err != nil {
						klog.Warningf("message: %s, update pod status failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, getPod.Namespace, getPod.Name)
					} else {
						klog.V(5).Infof("message: %s, update pod status successfully, namespace: %s, name: %s", msg.GetID(), updatedPod.Namespace, updatedPod.Name)

						// send response message to edged
						uc.podStatusResponse(msg)

						if updatedPod.DeletionTimestamp != nil && (status.Phase == v1.PodSucceeded || status.Phase == v1.PodFailed) {
							if uc.isPodNotRunning(status.ContainerStatuses) {
								if err := uc.kubeClient.CoreV1().Pods(updatedPod.Namespace).Delete(context.Background(), updatedPod.Name, *metaV1.NewDeleteOptions(0)); err != nil {
									klog.Warningf("message: %s, graceful delete pod failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, updatedPod.Namespace, updatedPod.Name)
								} else {
									klog.Infof("message: %s, pod delete successfully, namespace: %s, name: %s", msg.GetID(), updatedPod.Namespace, updatedPod.Name)
								}
							}
						}
					}
				}

			default:
				klog.Warningf("message: %s process failure, pod status operation: %s unsupported", msg.GetID(), msg.GetOperation())
				continue
			}
			klog.V(4).Infof("message: %s process successfully", msg.GetID())
		}
	}
}

// createNode create new edge node to kubernetes
func (uc *UpstreamController) createNode(name string, node *v1.Node) (*v1.Node, error) {
	node.Name = name
	return uc.kubeClient.CoreV1().Nodes().Create(context.Background(), node, metaV1.CreateOptions{})
}

// updateNodeStatus update node status
// Deprecated: updateNodeStatus will be deleted in subsequent versions, use patchNode instead.
func (uc *UpstreamController) updateNodeStatus() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop updateNodeStatus")
			return
		case msg := <-uc.nodeStatusChan:
			klog.V(5).Infof("message: %s, operation is: %s, and resource is %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

			data, err := msg.GetContentData()
			if err != nil {
				klog.Warningf("message: %s process failure, get content data failed with error: %s", msg.GetID(), err)
				continue
			}

			namespace, err := messagelayer.GetNamespace(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get namespace failed with error: %s", msg.GetID(), err)
				continue
			}
			name, err := messagelayer.GetResourceName(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get resource name failed with error: %s", msg.GetID(), err)
				continue
			}

			switch msg.GetOperation() {
			case model.InsertOperation:
				_, err := uc.kubeClient.CoreV1().Nodes().Get(context.Background(), name, metaV1.GetOptions{})
				if err == nil {
					klog.Infof("node: %s already exists, do nothing", name)
					uc.nodeMsgResponse(name, namespace, common.MessageSuccessfulContent, msg)
					continue
				}

				if !errors.IsNotFound(err) {
					errLog := fmt.Sprintf("get node %s info error: %v , register node failed", name, err)
					klog.Error(errLog)
					uc.nodeMsgResponse(name, namespace, errLog, msg)
					continue
				}

				node := &v1.Node{}
				err = json.Unmarshal(data, node)
				if err != nil {
					errLog := fmt.Sprintf("message: %s process failure, unmarshal marshaled message content with error: %s", msg.GetID(), err)
					klog.Error(errLog)
					uc.nodeMsgResponse(name, namespace, errLog, msg)
					continue
				}

				if _, err = uc.createNode(name, node); err != nil {
					errLog := fmt.Sprintf("create node %s error: %v , register node failed", name, err)
					klog.Error(errLog)
					uc.nodeMsgResponse(name, namespace, errLog, msg)
					continue
				}

				uc.nodeMsgResponse(name, namespace, common.MessageSuccessfulContent, msg)

			case model.UpdateOperation:
				nodeStatusRequest := &edgeapi.NodeStatusRequest{}
				err := json.Unmarshal(data, nodeStatusRequest)
				if err != nil {
					klog.Warningf("message: %s process failure, unmarshal marshaled message content with error: %s", msg.GetID(), err)
					continue
				}

				getNode, err := uc.kubeClient.CoreV1().Nodes().Get(context.Background(), name, metaV1.GetOptions{})
				if errors.IsNotFound(err) {
					klog.Warningf("message: %s process failure, node %s not found", msg.GetID(), name)
					continue
				}

				if err != nil {
					klog.Warningf("message: %s process failure with error: %s, namespaces: %s name: %s", msg.GetID(), err, namespace, name)
					continue
				}

				// TODO: comment below for test failure. Needs to decide whether to keep post troubleshoot
				// In case the status stored at metadata service is outdated, update the heartbeat automatically
				for i := range nodeStatusRequest.Status.Conditions {
					if time.Since(nodeStatusRequest.Status.Conditions[i].LastHeartbeatTime.Time) > time.Duration(uc.config.NodeUpdateFrequency)*time.Second {
						nodeStatusRequest.Status.Conditions[i].LastHeartbeatTime = metaV1.NewTime(time.Now())
					}
				}

				if getNode.Annotations == nil {
					getNode.Annotations = make(map[string]string)
				}
				for name, v := range nodeStatusRequest.ExtendResources {
					if name == constants.NvidiaGPUScalarResourceName {
						var gpuStatus []types.NvidiaGPUStatus
						for _, er := range v {
							gpuStatus = append(gpuStatus, types.NvidiaGPUStatus{ID: er.Name, Healthy: true})
						}
						if len(gpuStatus) > 0 {
							data, _ := json.Marshal(gpuStatus)
							getNode.Annotations[constants.NvidiaGPUStatusAnnotationKey] = string(data)
						}
					}
					data, err := json.Marshal(v)
					if err != nil {
						klog.Warningf("message: %s process failure, extend resource list marshal with error: %s", msg.GetID(), err)
						continue
					}
					getNode.Annotations[string(name)] = string(data)
				}

				// Keep the same "VolumesAttached" attribute with upstream,
				// since this value is maintained by kube-controller-manager.
				nodeStatusRequest.Status.VolumesAttached = getNode.Status.VolumesAttached
				if getNode.Status.DaemonEndpoints.KubeletEndpoint.Port != 0 {
					nodeStatusRequest.Status.DaemonEndpoints.KubeletEndpoint.Port = getNode.Status.DaemonEndpoints.KubeletEndpoint.Port
				}

				getNode.Status = nodeStatusRequest.Status

				node, err := uc.kubeClient.CoreV1().Nodes().UpdateStatus(context.Background(), getNode, metaV1.UpdateOptions{})
				if err != nil {
					klog.Warningf("message: %s process failure, update node failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, getNode.Namespace, getNode.Name)
					continue
				}

				nodeID, err := messagelayer.GetNodeID(msg)
				if err != nil {
					klog.Warningf("Message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
					continue
				}

				resource, err := messagelayer.BuildResource(nodeID, namespace, model.ResourceTypeNode, name)
				if err != nil {
					klog.Warningf("Message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
					continue
				}

				resMsg := model.NewMessage(msg.GetID()).
					SetResourceVersion(node.ResourceVersion).
					FillBody(common.MessageSuccessfulContent).
					BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				if err = uc.messageLayer.Response(*resMsg); err != nil {
					klog.Warningf("Message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}

				klog.V(4).Infof("message: %s, update node status successfully, namespace: %s, name: %s", msg.GetID(), getNode.Namespace, getNode.Name)

			default:
				klog.Warningf("message: %s process failure, node status operation: %s unsupported", msg.GetID(), msg.GetOperation())
				continue
			}
			klog.V(4).Infof("message: %s process successfully", msg.GetID())
		}
	}
}

func kubeClientGet(uc *UpstreamController, namespace string, name string, queryType string, msg model.Message) (metaV1.Object, error) {
	var obj metaV1.Object
	var err error
	switch queryType {
	case model.ResourceTypeConfigmap:
		obj, err = uc.configMapLister.ConfigMaps(namespace).Get(name)
	case model.ResourceTypeSecret:
		obj, err = uc.secretLister.Secrets(namespace).Get(name)
	case common.ResourceTypePersistentVolume:
		obj, err = uc.kubeClient.CoreV1().PersistentVolumes().Get(context.Background(), name, metaV1.GetOptions{})
	case common.ResourceTypePersistentVolumeClaim:
		obj, err = uc.kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metaV1.GetOptions{})
	case common.ResourceTypeVolumeAttachment:
		obj, err = uc.kubeClient.StorageV1().VolumeAttachments().Get(context.Background(), name, metaV1.GetOptions{})
	case model.ResourceTypeNode:
		obj, err = uc.nodeLister.Get(name)
	case model.ResourceTypeServiceAccountToken:
		obj, err = uc.getServiceAccountToken(namespace, name, msg)
	case model.ResourceTypeLease:
		obj, err = uc.leaseLister.Leases(namespace).Get(name)
	default:
		err = stderrors.New("wrong query type")
	}
	if err != nil {
		return nil, err
	}
	if err := util.SetMetaType(obj.(runtime.Object)); err != nil {
		return nil, err
	}
	return obj, nil
}

func queryInner(uc *UpstreamController, msg model.Message, queryType string) {
	klog.V(4).Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
	namespace, err := messagelayer.GetNamespace(msg)
	if err != nil {
		klog.Warningf("message: %s process failure, get namespace failed with error: %s", msg.GetID(), err)
		return
	}
	name, err := messagelayer.GetResourceName(msg)
	if err != nil {
		klog.Warningf("message: %s process failure, get resource name failed with error: %s", msg.GetID(), err)
		return
	}

	switch msg.GetOperation() {
	case model.QueryOperation:
		object, err := kubeClientGet(uc, namespace, name, queryType, msg)
		if errors.IsNotFound(err) {
			klog.Warningf("message: %s process failure, resource not found, namespace: %s, name: %s", msg.GetID(), namespace, name)
			return
		}
		if err != nil {
			klog.Warningf("message: %s process failure with error: %s, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
			return
		}

		nodeID, err := messagelayer.GetNodeID(msg)
		if err != nil {
			klog.Warningf("message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
			return
		}
		resource, err := messagelayer.BuildResource(nodeID, namespace, queryType, name)
		if err != nil {
			klog.Warningf("message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
			return
		}

		resMsg := model.NewMessage(msg.GetID()).
			SetResourceVersion(object.GetResourceVersion()).
			FillBody(object).
			BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
		err = uc.messageLayer.Response(*resMsg)
		if err != nil {
			klog.Warningf("message: %s process failure, response failed with error: %s", msg.GetID(), err)
			return
		}
		klog.V(4).Infof("message: %s process successfully", msg.GetID())
	default:
		klog.Warningf("message: %s process failure, operation: %s unsupported", msg.GetID(), msg.GetOperation())
	}
}

func (uc *UpstreamController) queryConfigMap() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop queryConfigMap")
			return
		case msg := <-uc.configMapChan:
			queryInner(uc, msg, model.ResourceTypeConfigmap)
		}
	}
}

func (uc *UpstreamController) querySecret() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop querySecret")
			return
		case msg := <-uc.secretChan:
			queryInner(uc, msg, model.ResourceTypeSecret)
		}
	}
}

func (uc *UpstreamController) processServiceAccountToken() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop process service account token")
			return
		case msg := <-uc.serviceAccountTokenChan:
			queryInner(uc, msg, model.ResourceTypeServiceAccountToken)
		}
	}
}

func (uc *UpstreamController) getServiceAccountToken(namespace string, name string, msg model.Message) (metaV1.Object, error) {
	data, err := msg.GetContentData()
	if err != nil {
		klog.Errorf("get message body failed err %v", err)
		return nil, err
	}

	tr := authenticationv1.TokenRequest{}
	if err := json.Unmarshal(data, &tr); err != nil {
		klog.Errorf("unmarshal token request failed err %v", err)
		return nil, err
	}

	tokenRequest, err := uc.kubeClient.CoreV1().ServiceAccounts(namespace).CreateToken(context.TODO(), name, &tr, metaV1.CreateOptions{})
	if err != nil {
		klog.Errorf("apiserver get service account token failed: err %v", err)
		return nil, err
	}

	return tokenRequest, nil
}

func (uc *UpstreamController) queryPersistentVolume() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop queryPersistentVolume")
			return
		case msg := <-uc.persistentVolumeChan:
			queryInner(uc, msg, common.ResourceTypePersistentVolume)
		}
	}
}

func (uc *UpstreamController) queryPersistentVolumeClaim() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop queryPersistentVolumeClaim")
			return
		case msg := <-uc.persistentVolumeClaimChan:
			queryInner(uc, msg, common.ResourceTypePersistentVolumeClaim)
		}
	}
}

func (uc *UpstreamController) queryVolumeAttachment() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop queryVolumeAttachment")
			return
		case msg := <-uc.volumeAttachmentChan:
			queryInner(uc, msg, common.ResourceTypeVolumeAttachment)
		}
	}
}

func (uc *UpstreamController) registerNode() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop registerNode")
			return
		case msg := <-uc.createNodeChan:
			klog.V(5).Infof("message: %s, operation is: %s, and resource is %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

			data, err := msg.GetContentData()
			if err != nil {
				klog.Warningf("message: %s process failure, get content data failed with error: %v", msg.GetID(), err)
				continue
			}

			namespace, err := messagelayer.GetNamespace(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get namespace failed with error: %v", msg.GetID(), err)
				continue
			}
			name, err := messagelayer.GetResourceName(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get resource name failed with error: %v", msg.GetID(), err)
				continue
			}

			node := &v1.Node{}
			err = json.Unmarshal(data, node)
			if err != nil {
				errLog := fmt.Sprintf("message: %s process failure, unmarshal marshaled message content with error: %v", msg.GetID(), err)
				klog.Error(errLog)
				uc.nodeMsgResponse(name, namespace, errLog, msg)
				continue
			}

			resp, err := uc.createNode(name, node)
			if err != nil {
				klog.Errorf("create node %s error: %v , register node failed", name, err)
			}

			resMsg := model.NewMessage(msg.GetID()).
				FillBody(&ObjectResp{Object: resp, Err: err}).
				BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, msg.GetResource(), model.ResponseOperation)
			if err = uc.messageLayer.Response(*resMsg); err != nil {
				klog.Warningf("Response message: %s failed, response failed with error: %v", msg.GetID(), err)
				continue
			}

			klog.V(4).Infof("message: %s, register node successfully, namespace: %s, name: %s", msg.GetID(), namespace, name)
		}
	}
}

func (uc *UpstreamController) patchNode() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop patchNode")
			return
		case msg := <-uc.patchNodeChan:
			klog.V(5).Infof("message: %s, operation is: %s, and resource is %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

			namespace, err := messagelayer.GetNamespace(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get namespace failed with error: %v", msg.GetID(), err)
				continue
			}
			name, err := messagelayer.GetResourceName(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get resource name failed with error: %v", msg.GetID(), err)
				continue
			}

			dataReq, ok := msg.Content.(string)
			if !ok {
				klog.Warningf("message: %s process failure, get data failed with error: %v", msg.GetID(), err)
				continue
			}

			nodeByte, err := base64.URLEncoding.DecodeString(dataReq)
			if err != nil {
				klog.Warningf("message: %s process failure, decode content data failed with error: %v", msg.GetID(), err)
				continue
			}

			node, err := uc.kubeClient.CoreV1().Nodes().Patch(context.TODO(), name, apimachineryType.StrategicMergePatchType, nodeByte, metaV1.PatchOptions{}, "status")
			if err != nil {
				klog.Errorf("message: %s process failure, patch node failed with error: %v, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
			}

			resMsg := model.NewMessage(msg.GetID()).
				SetResourceVersion(node.ResourceVersion).
				FillBody(&ObjectResp{Object: node, Err: err}).
				BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, msg.GetResource(), model.ResponseOperation)
			if err = uc.messageLayer.Response(*resMsg); err != nil {
				klog.Warningf("Message: %s process failure, response failed with error: %v", msg.GetID(), err)
				continue
			}

			klog.V(4).Infof("message: %s, patch node status successfully, namespace: %s, name: %s", msg.GetID(), namespace, name)
		}
	}
}

func (uc *UpstreamController) updateNode() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop updateNode")
			return
		case msg := <-uc.updateNodeChan:
			klog.V(5).Infof("message: %s, operation is: %s, and resource is %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
			noderequest := &v1.Node{}

			data, err := msg.GetContentData()
			if err != nil {
				klog.Warningf("message: %s process failure, get content data failed with error: %s", msg.GetID(), err)
				continue
			}

			if err := json.Unmarshal(data, noderequest); err != nil {
				klog.Warningf("message: %s process failure, unmarshal message content data with error: %s", msg.GetID(), err)
				continue
			}

			namespace, err := messagelayer.GetNamespace(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get namespace failed with error: %s", msg.GetID(), err)
				continue
			}
			name, err := messagelayer.GetResourceName(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get resource name failed with error: %s", msg.GetID(), err)
				continue
			}

			switch msg.GetOperation() {
			case model.UpdateOperation:
				getNode, err := uc.kubeClient.CoreV1().Nodes().Get(context.Background(), name, metaV1.GetOptions{})
				if errors.IsNotFound(err) {
					klog.Warningf("message: %s process failure, node %s not found", msg.GetID(), name)
					continue
				}
				if err != nil {
					klog.Warningf("message: %s process failure with error: %s, name: %s", msg.GetID(), err, name)
					continue
				}
				// update node labels
				if getNode.Labels == nil {
					getNode.Labels = make(map[string]string)
				}
				for key, value := range noderequest.Labels {
					getNode.Labels[key] = value
				}

				if getNode.Annotations == nil {
					getNode.Annotations = make(map[string]string)
				}
				for k, v := range noderequest.Annotations {
					getNode.Annotations[k] = v
				}
				byteNode, err := json.Marshal(getNode)
				if err != nil {
					klog.Warningf("marshal node data failed with err: %s", err)
					continue
				}
				node, err := uc.kubeClient.CoreV1().Nodes().Patch(context.Background(), getNode.Name, apimachineryType.StrategicMergePatchType, byteNode, metaV1.PatchOptions{})
				if err != nil {
					klog.Warningf("message: %s process failure, update node failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, getNode.Namespace, getNode.Name)
					continue
				}

				nodeID, err := messagelayer.GetNodeID(msg)
				if err != nil {
					klog.Warningf("Message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
					continue
				}
				resource, err := messagelayer.BuildResource(nodeID, namespace, model.ResourceTypeNode, name)
				if err != nil {
					klog.Warningf("Message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
					continue
				}

				resMsg := model.NewMessage(msg.GetID()).
					SetResourceVersion(node.ResourceVersion).
					FillBody(common.MessageSuccessfulContent).
					BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				if err = uc.messageLayer.Response(*resMsg); err != nil {
					klog.Warningf("Message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}

				klog.V(4).Infof("message: %s, update node successfully, namespace: %s, name: %s", msg.GetID(), getNode.Namespace, getNode.Name)
			default:
				klog.Warningf("message: %s process failure, node operation: %s unsupported", msg.GetID(), msg.GetOperation())
				continue
			}
			klog.V(4).Infof("message: %s process successfully", msg.GetID())
		}
	}
}

func (uc *UpstreamController) patchPod() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop patchPod")
			return
		case msg := <-uc.patchPodChan:
			klog.V(5).Infof("message: %s, operation is: %s, and resource is %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

			namespace, err := messagelayer.GetNamespace(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get namespace failed with error: %v", msg.GetID(), err)
				continue
			}
			name, err := messagelayer.GetResourceName(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get resource name failed with error: %v", msg.GetID(), err)
				continue
			}

			dataReq, ok := msg.Content.(string)
			if !ok {
				klog.Warningf("message: %s process failure, get data failed with error: %v", msg.GetID(), err)
				continue
			}

			podByte, err := base64.URLEncoding.DecodeString(dataReq)
			if err != nil {
				klog.Warningf("message: %s process failure, decode content data failed with error: %v", msg.GetID(), err)
				continue
			}

			updatedPod, err := uc.kubeClient.CoreV1().Pods(namespace).Patch(context.TODO(), name, apimachineryType.StrategicMergePatchType, podByte, metaV1.PatchOptions{}, "status")
			if err != nil {
				klog.Errorf("message: %s process failure, patch pod failed with error: %v, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
			}

			resMsg := model.NewMessage(msg.GetID()).
				SetResourceVersion(updatedPod.ResourceVersion).
				FillBody(&ObjectResp{Object: updatedPod, Err: err}).
				BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, msg.GetResource(), model.ResponseOperation)
			if err = uc.messageLayer.Response(*resMsg); err != nil {
				klog.Errorf("Message: %s process failure, response failed with error: %v", msg.GetID(), err)
				continue
			}

			klog.V(4).Infof("message: %s, patch pod successfully, namespace: %s, name: %s", msg.GetID(), namespace, name)
		}
	}
}

func (uc *UpstreamController) deletePod() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop deletePod")
			return
		case msg := <-uc.podDeleteChan:
			klog.V(5).Infof("message: %s, operation is: %s, and resource is %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

			namespace, err := messagelayer.GetNamespace(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get namespace failed with error: %s", msg.GetID(), err)
				continue
			}
			name, err := messagelayer.GetResourceName(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get resource name failed with error: %s", msg.GetID(), err)
				continue
			}

			podUID, ok := msg.Content.(string)
			if !ok {
				klog.Warningf("Failed to get podUID from msg, pod namesapce: %s, pod name: %s", namespace, name)
				continue
			}

			deleteOptions := metaV1.NewDeleteOptions(0)
			// Use the pod UID as the precondition for deletion to prevent deleting a newly created pod with the same name and namespace.
			deleteOptions.Preconditions = metaV1.NewUIDPreconditions(podUID)
			err = uc.kubeClient.CoreV1().Pods(namespace).Delete(context.Background(), name, *deleteOptions)
			if err != nil {
				klog.Warningf("Failed to delete pod, namespace: %s, name: %s, err: %v", namespace, name, err)
				continue
			}
			klog.V(4).Infof("Successfully terminate and remove pod from etcd, namespace: %s, name: %s", namespace, name)
		}
	}
}

func (uc *UpstreamController) queryNode() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop queryNode")
			return
		case msg := <-uc.queryNodeChan:
			queryInner(uc, msg, model.ResourceTypeNode)
		}
	}
}

func (uc *UpstreamController) createOrUpdateLease() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop create or update lease")
			return
		case msg := <-uc.createLeaseChan:
			klog.V(4).Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

			data, err := msg.GetContentData()
			if err != nil {
				klog.Warningf("message: %s process failure, get content data failed with error: %v", msg.GetID(), err)
				continue
			}

			namespace, err := messagelayer.GetNamespace(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get namespace failed with error: %v", msg.GetID(), err)
				return
			}
			name, err := messagelayer.GetResourceName(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get resource name failed with error: %v", msg.GetID(), err)
				continue
			}

			lease := &coordinationv1.Lease{}
			err = json.Unmarshal(data, lease)
			if err != nil {
				errLog := fmt.Sprintf("message: %s process failure, unmarshal message content with error: %v", msg.GetID(), err)
				klog.Error(errLog)
				uc.nodeMsgResponse(name, namespace, errLog, msg)
				continue
			}

			switch msg.GetOperation() {
			case model.InsertOperation:
				resp, err := uc.kubeClient.CoordinationV1().Leases(namespace).Create(context.TODO(), lease, metaV1.CreateOptions{})
				if err != nil {
					klog.Errorf("create lease %s failed, error: %v", name, err)
				}

				resMsg := model.NewMessage(msg.GetID()).
					FillBody(&ObjectResp{Object: resp, Err: err}).
					BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, msg.GetResource(), model.ResponseOperation)
				if err = uc.messageLayer.Response(*resMsg); err != nil {
					klog.Warningf("Response message: %s failed, response failed with error: %v", msg.GetID(), err)
					continue
				}

				klog.V(4).Infof("message: %s, create lease successfully, namespace: %s, name: %s", msg.GetID(), namespace, name)

			case model.UpdateOperation:
				resp, err := uc.kubeClient.CoordinationV1().Leases(namespace).Update(context.TODO(), lease, metaV1.UpdateOptions{})
				if err != nil {
					klog.Error("Update lease %s failed, error: %v", name, err)
				}

				resMsg := model.NewMessage(msg.GetID()).
					FillBody(&ObjectResp{Object: resp, Err: err}).
					BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, msg.GetResource(), model.ResponseOperation)
				if err = uc.messageLayer.Response(*resMsg); err != nil {
					klog.Warningf("Response message: %s failed, response failed with error: %v", msg.GetID(), err)
					continue
				}

				klog.V(4).Infof("message: %s, update lease successfully, namespace: %s, name: %s", msg.GetID(), namespace, name)

			default:
				klog.Warningf("message: %s process failure, operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
		}
	}
}

func (uc *UpstreamController) queryLease() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop queryLease")
			return
		case msg := <-uc.queryLeaseChan:
			klog.V(4).Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
			namespace, err := messagelayer.GetNamespace(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get namespace failed with error: %v", msg.GetID(), err)
				return
			}
			name, err := messagelayer.GetResourceName(msg)
			if err != nil {
				klog.Warningf("message: %s process failure, get resource name failed with error: %v", msg.GetID(), err)
				return
			}

			object, err := kubeClientGet(uc, namespace, name, model.ResourceTypeLease, msg)
			if err != nil {
				klog.Error("Query lease %s failed, error: %v", name, err)
			}

			resMsg := model.NewMessage(msg.GetID()).
				FillBody(&ObjectResp{Object: object, Err: err}).
				BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, msg.GetResource(), model.ResponseOperation)
			if err = uc.messageLayer.Response(*resMsg); err != nil {
				klog.Warningf("Response message: %s failed, response failed with error: %v", msg.GetID(), err)
				continue
			}

			klog.V(4).Infof("message: %s, query lease successfully, namespace: %s, name: %s", msg.GetID(), namespace, name)
		}
	}
}

func (uc *UpstreamController) unmarshalPodStatusMessage(msg model.Message) (ns string, podStatuses []edgeapi.PodStatusRequest) {
	ns, err := messagelayer.GetNamespace(msg)
	if err != nil {
		klog.Warningf("message: %s process failure, get namespace with error: %s", msg.GetID(), err)
		return
	}

	data, err := msg.GetContentData()
	if err != nil {
		klog.Warningf("message: %s process failure, get content data failed with error: %s", msg.GetID(), err)
		return
	}

	if name, _ := messagelayer.GetResourceName(msg); name == "" {
		// multi pod status in one message
		_ = json.Unmarshal(data, &podStatuses)
		return
	}

	// one pod status per message
	var status edgeapi.PodStatusRequest
	if err := json.Unmarshal(data, &status); err != nil {
		return
	}
	podStatuses = append(podStatuses, status)
	return
}

// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil if the condition is not present, or return the located condition.
func (uc *UpstreamController) getPodCondition(status *v1.PodStatus, conditionType v1.PodConditionType) *v1.PodCondition {
	if status == nil {
		return nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return &status.Conditions[i]
		}
	}
	return nil
}

func (uc *UpstreamController) isPodNotRunning(statuses []v1.ContainerStatus) bool {
	for _, status := range statuses {
		if status.State.Terminated == nil && status.State.Waiting == nil {
			return false
		}
	}
	return true
}

// We add this function, because apiserver only supports *RFC3339* now, which means that the timestamp returned by
// apiserver has no nanosecond information. However, the timestamp returned by unversioned.Now() contains nanosecond,
// so when we do comparison between status from apiserver and cached status, isStatusEqual() will always return false.
// There is related issue #15262 and PR #15263 about this.
func (uc *UpstreamController) normalizePodStatus(pod *v1.Pod, status *v1.PodStatus) *v1.PodStatus {
	normalizeTimeStamp := func(t *metaV1.Time) {
		*t = t.Rfc3339Copy()
	}
	normalizeContainerState := func(c *v1.ContainerState) {
		if c.Running != nil {
			normalizeTimeStamp(&c.Running.StartedAt)
		}
		if c.Terminated != nil {
			normalizeTimeStamp(&c.Terminated.StartedAt)
			normalizeTimeStamp(&c.Terminated.FinishedAt)
		}
	}

	if status.StartTime != nil {
		normalizeTimeStamp(status.StartTime)
	}
	for i := range status.Conditions {
		condition := &status.Conditions[i]
		normalizeTimeStamp(&condition.LastProbeTime)
		normalizeTimeStamp(&condition.LastTransitionTime)
	}

	// update container statuses
	for i := range status.ContainerStatuses {
		cstatus := &status.ContainerStatuses[i]
		normalizeContainerState(&cstatus.State)
		normalizeContainerState(&cstatus.LastTerminationState)
	}
	// Sort the container statuses, so that the order won't affect the result of comparison
	sort.Sort(SortedContainerStatuses(status.ContainerStatuses))

	// update init container statuses
	for i := range status.InitContainerStatuses {
		cstatus := &status.InitContainerStatuses[i]
		normalizeContainerState(&cstatus.State)
		normalizeContainerState(&cstatus.LastTerminationState)
	}
	// Sort the container statuses, so that the order won't affect the result of comparison
	SortInitContainerStatuses(pod, status.InitContainerStatuses)
	return status
}

// nodeMsgResponse response message of ResourceTypeNode
func (uc *UpstreamController) nodeMsgResponse(nodeName, namespace, content string, msg model.Message) {
	nodeID, err := messagelayer.GetNodeID(msg)
	if err != nil {
		klog.Warningf("Response message: %s failed, get node: %s id failed with error: %s", msg.GetID(), nodeName, err)
		return
	}

	resource, err := messagelayer.BuildResource(nodeID, namespace, model.ResourceTypeNode, nodeName)
	if err != nil {
		klog.Warningf("Response message: %s failed, build message resource failed with error: %s", msg.GetID(), err)
		return
	}

	resMsg := model.NewMessage(msg.GetID()).
		FillBody(content).
		BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
	if err = uc.messageLayer.Response(*resMsg); err != nil {
		klog.Warningf("Response message: %s failed, response failed with error: %s", msg.GetID(), err)
		return
	}
}

// NewUpstreamController create UpstreamController from config
func NewUpstreamController(config *v1alpha1.EdgeController, factory k8sinformer.SharedInformerFactory) (*UpstreamController, error) {
	uc := &UpstreamController{
		kubeClient:   client.GetKubeClient(),
		messageLayer: messagelayer.EdgeControllerMessageLayer(),
		crdClient:    client.GetCRDClient(),
		config:       *config,
	}
	uc.nodeLister = factory.Core().V1().Nodes().Lister()
	uc.endpointLister = factory.Core().V1().Endpoints().Lister()
	uc.serviceLister = factory.Core().V1().Services().Lister()
	uc.podLister = factory.Core().V1().Pods().Lister()
	uc.configMapLister = factory.Core().V1().ConfigMaps().Lister()
	uc.secretLister = factory.Core().V1().Secrets().Lister()
	uc.leaseLister = factory.Coordination().V1().Leases().Lister()

	uc.nodeStatusChan = make(chan model.Message, config.Buffer.UpdateNodeStatus)
	uc.podStatusChan = make(chan model.Message, config.Buffer.UpdatePodStatus)
	uc.configMapChan = make(chan model.Message, config.Buffer.QueryConfigMap)
	uc.secretChan = make(chan model.Message, config.Buffer.QuerySecret)
	uc.serviceAccountTokenChan = make(chan model.Message, config.Buffer.ServiceAccountToken)
	uc.serviceChan = make(chan model.Message, config.Buffer.QueryService)
	uc.endpointsChan = make(chan model.Message, config.Buffer.QueryEndpoints)
	uc.persistentVolumeChan = make(chan model.Message, config.Buffer.QueryPersistentVolume)
	uc.persistentVolumeClaimChan = make(chan model.Message, config.Buffer.QueryPersistentVolumeClaim)
	uc.volumeAttachmentChan = make(chan model.Message, config.Buffer.QueryVolumeAttachment)
	uc.createNodeChan = make(chan model.Message, config.Buffer.CreateNode)
	uc.patchNodeChan = make(chan model.Message, config.Buffer.PatchNode)
	uc.queryNodeChan = make(chan model.Message, config.Buffer.QueryNode)
	uc.updateNodeChan = make(chan model.Message, config.Buffer.UpdateNode)
	uc.patchPodChan = make(chan model.Message, config.Buffer.PatchPod)
	uc.podDeleteChan = make(chan model.Message, config.Buffer.DeletePod)
	uc.createLeaseChan = make(chan model.Message, config.Buffer.CreateLease)
	uc.queryLeaseChan = make(chan model.Message, config.Buffer.QueryLease)
	uc.ruleStatusChan = make(chan model.Message, config.Buffer.UpdateNodeStatus)
	return uc, nil
}
