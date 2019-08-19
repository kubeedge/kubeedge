/*
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
	"encoding/json"
	"fmt"
	"sort"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/types"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/utils"
	common "github.com/kubeedge/kubeedge/common/constants"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
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

// UpstreamController subscribe messages from edge and sync to k8s api server
type UpstreamController struct {
	kubeClient   *kubernetes.Clientset
	messageLayer messagelayer.MessageLayer

	//stop channel
	stopDispatch                   chan struct{}
	stopUpdateNodeStatus           chan struct{}
	stopUpdatePodStatus            chan struct{}
	stopQueryConfigMap             chan struct{}
	stopQuerySecret                chan struct{}
	stopQueryService               chan struct{}
	stopQueryEndpoints             chan struct{}
	stopQueryPersistentVolume      chan struct{}
	stopQueryPersistentVolumeClaim chan struct{}
	stopQueryVolumeAttachment      chan struct{}
	stopQueryNode                  chan struct{}
	stopUpdateNode                 chan struct{}

	// message channel
	nodeStatusChan            chan model.Message
	podStatusChan             chan model.Message
	secretChan                chan model.Message
	configMapChan             chan model.Message
	serviceChan               chan model.Message
	endpointsChan             chan model.Message
	persistentVolumeChan      chan model.Message
	persistentVolumeClaimChan chan model.Message
	volumeAttachmentChan      chan model.Message
	queryNodeChan             chan model.Message
	updateNodeChan            chan model.Message
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	klog.Info("start upstream controller")
	uc.stopDispatch = make(chan struct{})
	uc.stopUpdateNodeStatus = make(chan struct{})
	uc.stopUpdatePodStatus = make(chan struct{})
	uc.stopQueryConfigMap = make(chan struct{})
	uc.stopQuerySecret = make(chan struct{})
	uc.stopQueryService = make(chan struct{})
	uc.stopQueryEndpoints = make(chan struct{})
	uc.stopQueryPersistentVolume = make(chan struct{})
	uc.stopQueryPersistentVolumeClaim = make(chan struct{})
	uc.stopQueryVolumeAttachment = make(chan struct{})
	uc.stopQueryNode = make(chan struct{})
	uc.stopUpdateNode = make(chan struct{})

	uc.nodeStatusChan = make(chan model.Message, config.UpdateNodeStatusBuffer)
	uc.podStatusChan = make(chan model.Message, config.UpdatePodStatusBuffer)
	uc.configMapChan = make(chan model.Message, config.QueryConfigMapBuffer)
	uc.secretChan = make(chan model.Message, config.QuerySecretBuffer)
	uc.serviceChan = make(chan model.Message, config.QueryServiceBuffer)
	uc.endpointsChan = make(chan model.Message, config.QueryEndpointsBuffer)
	uc.persistentVolumeChan = make(chan model.Message, config.QueryPersistentVolumeBuffer)
	uc.persistentVolumeClaimChan = make(chan model.Message, config.QueryPersistentVolumeClaimBuffer)
	uc.volumeAttachmentChan = make(chan model.Message, config.QueryVolumeAttachmentBuffer)
	uc.queryNodeChan = make(chan model.Message, config.QueryNodeBuffer)
	uc.updateNodeChan = make(chan model.Message, config.UpdateNodeBuffer)

	go uc.dispatchMessage(uc.stopDispatch)

	for i := 0; i < config.UpdateNodeStatusWorkers; i++ {
		go uc.updateNodeStatus(uc.stopUpdateNodeStatus)
	}
	for i := 0; i < config.UpdatePodStatusWorkers; i++ {
		go uc.updatePodStatus(uc.stopUpdatePodStatus)
	}
	for i := 0; i < config.QueryConfigMapWorkers; i++ {
		go uc.queryConfigMap(uc.stopQueryConfigMap)
	}
	for i := 0; i < config.QuerySecretWorkers; i++ {
		go uc.querySecret(uc.stopQuerySecret)
	}
	for i := 0; i < config.QueryServiceWorkers; i++ {
		go uc.queryService(uc.stopQueryService)
	}
	for i := 0; i < config.QueryEndpointsWorkers; i++ {
		go uc.queryEndpoints(uc.stopQueryEndpoints)
	}
	for i := 0; i < config.QueryPersistentVolumeWorkers; i++ {
		go uc.queryPersistentVolume(uc.stopQueryPersistentVolume)
	}
	for i := 0; i < config.QueryPersistentVolumeClaimWorkers; i++ {
		go uc.queryPersistentVolumeClaim(uc.stopQueryPersistentVolumeClaim)
	}
	for i := 0; i < config.QueryVolumeAttachmentWorkers; i++ {
		go uc.queryVolumeAttachment(uc.stopQueryVolumeAttachment)
	}
	for i := 0; i < config.QueryNodeWorkers; i++ {
		go uc.queryNode(uc.stopQueryNode)
	}
	for i := 0; i < config.UpdateNodeBuffer; i++ {
		go uc.updateNode(uc.stopUpdateNode)
	}
	return nil
}

func (uc *UpstreamController) dispatchMessage(stop chan struct{}) {
	running := true
	go func() {
		<-stop
		klog.Info("stop dispatchMessage")
		running = false
	}()
	for running {
		msg, err := uc.messageLayer.Receive()
		if err != nil {
			klog.Warningf("receive message failed, %s", err)
			continue
		}

		klog.Infof("dispatch message: %s", msg.GetID())

		resourceType, err := messagelayer.GetResourceType(msg)
		if err != nil {
			klog.Warningf("parse message: %s resource type with error: %s", msg.GetID(), err)
			continue
		}
		klog.Infof("message: %s, resource type is: %s", msg.GetID(), resourceType)
		operationType := msg.GetOperation()
		if err != nil {
			klog.Warningf("parse message: %s operation type with error: %s", msg.GetID(), err)
			continue
		}
		klog.Infof("message: %s, operation type is: %s", msg.GetID(), operationType)

		switch resourceType {
		case model.ResourceTypeNodeStatus:
			uc.nodeStatusChan <- msg
		case model.ResourceTypePodStatus:
			uc.podStatusChan <- msg
		case model.ResourceTypeConfigmap:
			uc.configMapChan <- msg
		case model.ResourceTypeSecret:
			uc.secretChan <- msg
		case common.ResourceTypeService:
			uc.serviceChan <- msg
		case common.ResourceTypeEndpoints:
			uc.endpointsChan <- msg
		case common.ResourceTypePersistentVolume:
			uc.persistentVolumeChan <- msg
		case common.ResourceTypePersistentVolumeClaim:
			uc.persistentVolumeClaimChan <- msg
		case common.ResourceTypeVolumeAttachment:
			uc.volumeAttachmentChan <- msg
		case model.ResourceTypeNode:
			switch operationType {
			case model.QueryOperation:
				uc.queryNodeChan <- msg
			case model.UpdateOperation:
				uc.updateNodeChan <- msg
			default:
				err = fmt.Errorf("message: %s, operation type: %s unsupported", msg.GetID(), operationType)
			}
		default:
			err = fmt.Errorf("message: %s, resource type: %s unsupported", msg.GetID(), resourceType)
		}
	}
}

func (uc *UpstreamController) updatePodStatus(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.podStatusChan:
			klog.Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

			namespace, podStatuses := uc.unmarshalPodStatusMessage(msg)
			switch msg.GetOperation() {
			case model.UpdateOperation:
				for _, podStatus := range podStatuses {
					getPod, err := uc.kubeClient.CoreV1().Pods(namespace).Get(podStatus.Name, metaV1.GetOptions{})
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
						delMsg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
						if err := uc.messageLayer.Send(*delMsg); err != nil {
							klog.Warningf("Send message failed with error: %s, operation: %s, resource: %s", err, delMsg.GetOperation(), delMsg.GetResource())
						} else {
							klog.Infof("Send message successfully, operation: %s, resource: %s", delMsg.GetOperation(), delMsg.GetResource())
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
					if _, readyCondition := uc.getPodCondition(&status, v1.PodReady); readyCondition != nil {
						// Need to set LastTransitionTime.
						lastTransitionTime := metaV1.Now()
						_, oldReadyCondition := uc.getPodCondition(&oldStatus, v1.PodReady)
						if oldReadyCondition != nil && readyCondition.Status == oldReadyCondition.Status {
							lastTransitionTime = oldReadyCondition.LastTransitionTime
						}
						readyCondition.LastTransitionTime = lastTransitionTime
					}

					// Set InitializedCondition.LastTransitionTime.
					if _, initCondition := uc.getPodCondition(&status, v1.PodInitialized); initCondition != nil {
						// Need to set LastTransitionTime.
						lastTransitionTime := metaV1.Now()
						_, oldInitCondition := uc.getPodCondition(&oldStatus, v1.PodInitialized)
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

					if updatedPod, err := uc.kubeClient.CoreV1().Pods(getPod.Namespace).UpdateStatus(getPod); err != nil {
						klog.Warningf("message: %s, update pod status failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, getPod.Namespace, getPod.Name)
					} else {
						klog.Infof("message: %s, update pod status successfully, namespace: %s, name: %s", msg.GetID(), updatedPod.Namespace, updatedPod.Name)
						if updatedPod.DeletionTimestamp != nil && (status.Phase == v1.PodSucceeded || status.Phase == v1.PodFailed) {
							if uc.isPodNotRunning(status.ContainerStatuses) {
								if err := uc.kubeClient.CoreV1().Pods(updatedPod.Namespace).Delete(updatedPod.Name, metaV1.NewDeleteOptions(0)); err != nil {
									klog.Warningf("message: %s, graceful delete pod failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, updatedPod.Namespace, updatedPod.Name)
								}
								klog.Infof("message: %s, pod delete successfully, namespace: %s, name: %s", msg.GetID(), updatedPod.Namespace, updatedPod.Name)
							}
						}
					}
				}

			default:
				klog.Infof("pod status operation: %s unsupported", msg.GetOperation())
			}
			klog.Infof("message: %s process successfully", msg.GetID())
		case <-stop:
			klog.Info("stop updatePodStatus")
			running = false
		}
	}
}

func (uc *UpstreamController) updateNodeStatus(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.nodeStatusChan:
			klog.Infof("message: %s, operation is: %s, and resource is %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
			nodeStatusRequest := &edgeapi.NodeStatusRequest{}

			var data []byte
			switch msg.Content.(type) {
			case []byte:
				data = msg.GetContent().([]byte)
			default:
				var err error
				data, err = json.Marshal(msg.GetContent())
				if err != nil {
					klog.Warningf("message: %s process failure, marshal message content with error: %s", msg.GetID(), err)
					continue
				}
			}

			err := json.Unmarshal(data, nodeStatusRequest)
			if err != nil {
				klog.Warningf("message: %s process failure, unmarshal marshaled message content with error: %s", msg.GetID(), err)
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
				getNode, err := uc.kubeClient.CoreV1().Nodes().Get(name, metaV1.GetOptions{})
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
				if !config.EdgeSiteEnabled {
					for i := range nodeStatusRequest.Status.Conditions {
						if time.Now().Sub(nodeStatusRequest.Status.Conditions[i].LastHeartbeatTime.Time) > config.Kube.KubeUpdateNodeFrequency {
							nodeStatusRequest.Status.Conditions[i].LastHeartbeatTime = metaV1.NewTime(time.Now())
						}

						if time.Now().Sub(nodeStatusRequest.Status.Conditions[i].LastTransitionTime.Time) > config.Kube.KubeUpdateNodeFrequency {
							nodeStatusRequest.Status.Conditions[i].LastTransitionTime = metaV1.NewTime(time.Now())
						}
					}
				}

				if getNode.Annotations == nil {
					klog.Warningf("node annotations is nil map, new a map for it. namespace: %s, name: %s", getNode.Namespace, getNode.Name)
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

				getNode.Status = nodeStatusRequest.Status
				if _, err := uc.kubeClient.CoreV1().Nodes().UpdateStatus(getNode); err != nil {
					klog.Warningf("message: %s process failure, update node failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, getNode.Namespace, getNode.Name)
					continue
				}

				resMsg := model.NewMessage(msg.GetID())
				resMsg.Content = "OK"
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
				resMsg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				if err = uc.messageLayer.Response(*resMsg); err != nil {
					klog.Warningf("Message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}

				klog.Infof("message: %s, update node status successfully, namespace: %s, name: %s", msg.GetID(), getNode.Namespace, getNode.Name)

			default:
				klog.Infof("message: %s process failure, node status operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
			klog.Infof("message: %s process successfully", msg.GetID())
		case <-stop:
			klog.Info("stop updateNodeStatus")
			running = false
		}
	}
}

func (uc *UpstreamController) queryConfigMap(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.configMapChan:
			klog.Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
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
			case model.QueryOperation:
				configMap, err := uc.kubeClient.CoreV1().ConfigMaps(namespace).Get(name, metaV1.GetOptions{})
				if errors.IsNotFound(err) {
					klog.Warningf("message: %s process failure, config map not found, namespace: %s, name: %s", msg.GetID(), namespace, name)
					continue
				}
				if err != nil {
					klog.Warningf("message: %s process failure with error: %s, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
					continue
				}
				resMsg := model.NewMessage(msg.GetID())
				resMsg.Content = configMap
				nodeID, err := messagelayer.GetNodeID(msg)
				if err != nil {
					klog.Warningf("message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
				}
				resource, err := messagelayer.BuildResource(nodeID, configMap.Namespace, model.ResourceTypeConfigmap, configMap.Name)
				if err != nil {
					klog.Warningf("message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
				}
				resMsg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				err = uc.messageLayer.Response(*resMsg)
				if err != nil {
					klog.Warningf("message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}
				klog.Warningf("message: %s process successfully", msg.GetID())
			default:
				klog.Infof("message: %s process failure, config map operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
			klog.Infof("message: %s process successfully", msg.GetID())
		case <-stop:
			klog.Info("stop queryConfigMap")
			running = false
		}
	}
}

func (uc *UpstreamController) querySecret(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.secretChan:
			klog.Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
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
			case model.QueryOperation:
				secret, err := uc.kubeClient.CoreV1().Secrets(namespace).Get(name, metaV1.GetOptions{})
				if errors.IsNotFound(err) {
					klog.Warningf("message: %s process failure, secret not found, namespace: %s, name: %s", msg.GetID(), namespace, name)
					continue
				}
				if err != nil {
					klog.Warningf("message: %s process failure with error: %s, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
					continue
				}
				resMsg := model.NewMessage(msg.GetID())
				resMsg.Content = secret
				nodeID, err := messagelayer.GetNodeID(msg)
				if err != nil {
					klog.Warningf("message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
					continue
				}
				resource, err := messagelayer.BuildResource(nodeID, secret.Namespace, model.ResourceTypeSecret, secret.Name)
				if err != nil {
					klog.Warningf("message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
					continue
				}
				resMsg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				err = uc.messageLayer.Response(*resMsg)
				if err != nil {
					klog.Warningf("message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}
				klog.Warningf("message: %s process successfully", msg.GetID())
			default:
				klog.Infof("message: %s process failure, secret operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
			klog.Infof("message: %s process successfully", msg.GetID())
		case <-stop:
			klog.Info("stop querySecret")
			running = false
		}
	}
}

func (uc *UpstreamController) queryService(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.serviceChan:
			klog.Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
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
			case model.QueryOperation:
				svc, err := uc.kubeClient.CoreV1().Services(namespace).Get(name, metaV1.GetOptions{})
				if errors.IsNotFound(err) {
					klog.Warningf("message: %s process failure, service not found, namespace: %s, name: %s", msg.GetID(), namespace, name)
					continue
				}
				if err != nil {
					klog.Warningf("message: %s process failure with error: %s, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
					continue
				}
				resMsg := model.NewMessage(msg.GetID())
				resMsg.Content = svc
				nodeID, err := messagelayer.GetNodeID(msg)
				if err != nil {
					klog.Warningf("message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
				}
				resource, err := messagelayer.BuildResource(nodeID, svc.Namespace, common.ResourceTypeService, svc.Name)
				if err != nil {
					klog.Warningf("message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
				}
				resMsg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				err = uc.messageLayer.Response(*resMsg)
				if err != nil {
					klog.Warningf("message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}
				klog.Warningf("message: %s process successfully", msg.GetID())
			default:
				klog.Infof("message: %s process failure, service operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
		case <-stop:
			klog.Info("stop queryService")
			running = false
		}
	}
}

func (uc *UpstreamController) queryEndpoints(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.endpointsChan:
			klog.Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
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
			case model.QueryOperation:
				eps, err := uc.kubeClient.CoreV1().Endpoints(namespace).Get(name, metaV1.GetOptions{})
				if errors.IsNotFound(err) {
					klog.Warningf("message: %s process failure, endpoints not found, namespace: %s, name: %s", msg.GetID(), namespace, name)
					continue
				}
				if err != nil {
					klog.Warningf("message: %s process failure with error: %s, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
					continue
				}
				resMsg := model.NewMessage(msg.GetID())
				resMsg.Content = eps
				nodeID, err := messagelayer.GetNodeID(msg)
				if err != nil {
					klog.Warningf("message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
					continue
				}
				resource, err := messagelayer.BuildResource(nodeID, eps.Namespace, common.ResourceTypeEndpoints, eps.Name)
				if err != nil {
					klog.Warningf("message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
					continue
				}
				resMsg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				err = uc.messageLayer.Response(*resMsg)
				if err != nil {
					klog.Warningf("message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}
				klog.Warningf("message: %s process successfully", msg.GetID())
			default:
				klog.Infof("message: %s process failure, endpoints operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
			klog.Infof("message: %s process successfully", msg.GetID())
		case <-stop:
			klog.Info("stop queryEndpoints")
			running = false
		}
	}
}

func (uc *UpstreamController) queryPersistentVolume(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.persistentVolumeChan:
			klog.Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
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
			case model.QueryOperation:
				pv, err := uc.kubeClient.CoreV1().PersistentVolumes().Get(name, metaV1.GetOptions{})
				if errors.IsNotFound(err) {
					klog.Warningf("message: %s process failure, persistentvolume not found, namespace: %s, name: %s", msg.GetID(), namespace, name)
					continue
				}
				if err != nil {
					klog.Warningf("message: %s process failure with error: %s, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
					continue
				}
				resMsg := model.NewMessage(msg.GetID())
				resMsg.Content = pv
				nodeID, err := messagelayer.GetNodeID(msg)
				if err != nil {
					klog.Warningf("message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
					continue
				}
				resource, err := messagelayer.BuildResource(nodeID, namespace, "persistentvolume", pv.Name)
				if err != nil {
					klog.Warningf("message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
					continue
				}
				resMsg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				err = uc.messageLayer.Response(*resMsg)
				if err != nil {
					klog.Warningf("message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}
				klog.Warningf("message: %s process successfully", msg.GetID())
			default:
				klog.Infof("message: %s process failure, persistentvolume operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
			klog.Infof("message: %s process successfully", msg.GetID())
		case <-stop:
			klog.Infof("stop queryPersistentVolume")
			running = false
		}
	}
}

func (uc *UpstreamController) queryPersistentVolumeClaim(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.persistentVolumeClaimChan:
			klog.Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
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
			case model.QueryOperation:
				pvc, err := uc.kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(name, metaV1.GetOptions{})
				if errors.IsNotFound(err) {
					klog.Warningf("message: %s process failure, persistentvolumeclaim not found, namespace: %s, name: %s", msg.GetID(), namespace, name)
					continue
				}
				if err != nil {
					klog.Warningf("message: %s process failure with error: %s, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
					continue
				}
				resMsg := model.NewMessage(msg.GetID())
				resMsg.Content = pvc
				nodeID, err := messagelayer.GetNodeID(msg)
				if err != nil {
					klog.Warningf("message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
					continue
				}
				resource, err := messagelayer.BuildResource(nodeID, pvc.Namespace, "persistentvolumeclaim", pvc.Name)
				if err != nil {
					klog.Warningf("message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
					continue
				}
				resMsg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				err = uc.messageLayer.Response(*resMsg)
				if err != nil {
					klog.Warningf("message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}
				klog.Warningf("message: %s process successfully", msg.GetID())
			default:
				klog.Infof("message: %s process failure, persistentvolumeclaim operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
			klog.Infof("message: %s process successfully", msg.GetID())
		case <-stop:
			klog.Infof("stop queryPersistentVolumeClaim")
			running = false
		}
	}
}

func (uc *UpstreamController) queryVolumeAttachment(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.volumeAttachmentChan:
			klog.Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
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
			case model.QueryOperation:
				va, err := uc.kubeClient.StorageV1().VolumeAttachments().Get(name, metaV1.GetOptions{})
				if errors.IsNotFound(err) {
					klog.Warningf("message: %s process failure, volumeattachment not found, namespace: %s, name: %s", msg.GetID(), namespace, name)
					continue
				}
				if err != nil {
					klog.Warningf("message: %s process failure with error: %s, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
					continue
				}
				resMsg := model.NewMessage(msg.GetID())
				resMsg.Content = va
				nodeID, err := messagelayer.GetNodeID(msg)
				if err != nil {
					klog.Warningf("message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
					continue
				}
				resource, err := messagelayer.BuildResource(nodeID, namespace, "volumeattachment", va.Name)
				if err != nil {
					klog.Warningf("message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
					continue
				}
				resMsg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				err = uc.messageLayer.Response(*resMsg)
				if err != nil {
					klog.Warningf("message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}
				klog.Warningf("message: %s process successfully", msg.GetID())
			default:
				klog.Infof("message: %s process failure, volumeattachment operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
			klog.Infof("message: %s process successfully", msg.GetID())
		case <-stop:
			klog.Infof("stop queryVolumeAttachment")
			running = false
		}
	}
}

func (uc *UpstreamController) updateNode(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.updateNodeChan:
			klog.Infof("message: %s, operation is: %s, and resource is %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
			noderequest := &v1.Node{}

			var data []byte
			switch msg.Content.(type) {
			case []byte:
				data = msg.GetContent().([]byte)
			default:
				var err error
				data, err = json.Marshal(msg.GetContent())
				if err != nil {
					klog.Warningf("message: %s process failure, marshal message content with error: %s", msg.GetID(), err)
					continue
				}
			}

			err := json.Unmarshal(data, noderequest)
			if err != nil {
				klog.Warningf("message: %s process failure, unmarshal marshaled message content with error: %s", msg.GetID(), err)
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
				getNode, err := uc.kubeClient.CoreV1().Nodes().Get(name, metaV1.GetOptions{})
				if errors.IsNotFound(err) {
					klog.Warningf("message: %s process failure, node %s not found", msg.GetID(), name)
					continue
				}
				if err != nil {
					klog.Warningf("message: %s process failure with error: %s, name: %s", msg.GetID(), err, name)
					continue
				}

				if getNode.Annotations == nil {
					klog.Warningf("node annotations is nil map, new a map for it. namespace: %s, name: %s", getNode.Namespace, getNode.Name)
					getNode.Annotations = make(map[string]string)
				}
				for k, v := range noderequest.Annotations {
					getNode.Annotations[k] = v
				}

				if _, err := uc.kubeClient.CoreV1().Nodes().Update(getNode); err != nil {
					klog.Warningf("message: %s process failure, update node failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, getNode.Namespace, getNode.Name)
					continue
				}

				resMsg := model.NewMessage(msg.GetID())
				resMsg.Content = "OK"
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
				resMsg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				if err = uc.messageLayer.Response(*resMsg); err != nil {
					klog.Warningf("Message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}

				klog.Infof("message: %s, update node successfully, namespace: %s, name: %s", msg.GetID(), getNode.Namespace, getNode.Name)

			default:
				klog.Infof("message: %s process failure, node operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
			klog.Infof("message: %s process successfully", msg.GetID())
		case <-stop:
			klog.Infof("stop updateNode")
			running = false
		}
	}
}

func (uc *UpstreamController) queryNode(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.queryNodeChan:
			klog.Infof("message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
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
			case model.QueryOperation:
				node, err := uc.kubeClient.CoreV1().Nodes().Get(name, metaV1.GetOptions{})
				if errors.IsNotFound(err) {
					klog.Warningf("message: %s process failure, node not found, namespace: %s, name: %s", msg.GetID(), namespace, name)
					continue
				}
				if err != nil {
					klog.Warningf("message: %s process failure with error: %s, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
					continue
				}
				resMsg := model.NewMessage(msg.GetID())
				resMsg.Content = node
				nodeID, err := messagelayer.GetNodeID(msg)
				if err != nil {
					klog.Warningf("message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
				}
				resource, err := messagelayer.BuildResource(nodeID, namespace, model.ResourceTypeNode, node.Name)
				if err != nil {
					klog.Warningf("message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
				}
				resMsg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				err = uc.messageLayer.Response(*resMsg)
				if err != nil {
					klog.Warningf("message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}
				klog.Warningf("message: %s process successfully", msg.GetID())
			default:
				klog.Infof("message: %s process failure, query node operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
			klog.Infof("message: %s process successfully", msg.GetID())
		case <-stop:
			klog.Infof("stop queryNode")
			running = false
		}
	}
}

func (uc *UpstreamController) unmarshalPodStatusMessage(msg model.Message) (ns string, podStatuses []edgeapi.PodStatusRequest) {
	ns, err := messagelayer.GetNamespace(msg)
	if err != nil {
		klog.Warningf("message: %s process failure, get namespace with error: %s", msg.GetID(), err)
		return
	}
	name, _ := messagelayer.GetResourceName(msg)

	var data []byte
	switch msg.Content.(type) {
	case []byte:
		data = msg.GetContent().([]byte)
	default:
		var err error
		data, err = json.Marshal(msg.GetContent())
		if err != nil {
			klog.Warningf("message: %s process failure, marshal content failed with error: %s", msg.GetID(), err)
			return
		}
	}

	if name == "" {
		// multi pod status in one message
		err = json.Unmarshal(data, &podStatuses)
		if err != nil {
			return
		}
	} else {
		// one pod status per message
		var status edgeapi.PodStatusRequest
		if err := json.Unmarshal(data, &status); err != nil {
			return
		}
		podStatuses = append(podStatuses, status)
	}
	return
}

// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func (uc *UpstreamController) getPodCondition(status *v1.PodStatus, conditionType v1.PodConditionType) (int, *v1.PodCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
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

// Stop UpstreamController
func (uc *UpstreamController) Stop() error {
	klog.Info("Stopping upstream controller")
	defer klog.Info("Upstream controller stopped")

	uc.stopDispatch <- struct{}{}
	for i := 0; i < config.UpdateNodeStatusWorkers; i++ {
		uc.stopUpdateNodeStatus <- struct{}{}
	}
	for i := 0; i < config.UpdatePodStatusWorkers; i++ {
		uc.stopUpdatePodStatus <- struct{}{}
	}
	for i := 0; i < config.QueryConfigMapWorkers; i++ {
		uc.stopQueryConfigMap <- struct{}{}
	}
	for i := 0; i < config.QuerySecretWorkers; i++ {
		uc.stopQuerySecret <- struct{}{}
	}
	for i := 0; i < config.QueryServiceWorkers; i++ {
		uc.stopQueryService <- struct{}{}
	}
	for i := 0; i < config.QueryEndpointsWorkers; i++ {
		uc.stopQueryEndpoints <- struct{}{}
	}
	return nil
}

// NewUpstreamController create UpstreamController from config
func NewUpstreamController() (*UpstreamController, error) {
	cli, err := utils.KubeClient()
	if err != nil {
		klog.Warningf("create kube client failed with error: %s", err)
		return nil, err
	}
	ml, err := messagelayer.NewMessageLayer()
	if err != nil {
		klog.Warningf("create message layer failed with error: %s", err)
	}
	uc := &UpstreamController{kubeClient: cli, messageLayer: ml}
	return uc, nil
}
