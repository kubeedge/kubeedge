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
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"sort"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
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
	podDeleteChan             chan model.Message
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	klog.Info("start upstream controller")

	uc.nodeStatusChan = make(chan model.Message, config.Config.Buffer.UpdateNodeStatus)
	uc.podStatusChan = make(chan model.Message, config.Config.Buffer.UpdatePodStatus)
	uc.configMapChan = make(chan model.Message, config.Config.Buffer.QueryConfigMap)
	uc.secretChan = make(chan model.Message, config.Config.Buffer.QuerySecret)
	uc.serviceChan = make(chan model.Message, config.Config.Buffer.QueryService)
	uc.endpointsChan = make(chan model.Message, config.Config.Buffer.QueryEndpoints)
	uc.persistentVolumeChan = make(chan model.Message, config.Config.Buffer.QueryPersistentVolume)
	uc.persistentVolumeClaimChan = make(chan model.Message, config.Config.Buffer.QueryPersistentVolumeClaim)
	uc.volumeAttachmentChan = make(chan model.Message, config.Config.Buffer.QueryVolumeAttachment)
	uc.queryNodeChan = make(chan model.Message, config.Config.Buffer.QueryNode)
	uc.updateNodeChan = make(chan model.Message, config.Config.Buffer.UpdateNode)
	uc.podDeleteChan = make(chan model.Message, config.Config.Buffer.DeletePod)

	go uc.dispatchMessage()

	for i := 0; i < int(config.Config.Load.UpdateNodeStatusWorkers); i++ {
		go uc.updateNodeStatus()
	}
	for i := 0; i < int(config.Config.Load.UpdatePodStatusWorkers); i++ {
		go uc.updatePodStatus()
	}
	for i := 0; i < int(config.Config.Load.QueryConfigMapWorkers); i++ {
		go uc.queryConfigMap()
	}
	for i := 0; i < int(config.Config.Load.QuerySecretWorkers); i++ {
		go uc.querySecret()
	}
	for i := 0; i < int(config.Config.Load.QueryServiceWorkers); i++ {
		go uc.queryService()
	}
	for i := 0; i < int(config.Config.Load.QueryEndpointsWorkers); i++ {
		go uc.queryEndpoints()
	}
	for i := 0; i < int(config.Config.Load.QueryPersistentVolumeWorkers); i++ {
		go uc.queryPersistentVolume()
	}
	for i := 0; i < int(config.Config.Load.QueryPersistentVolumeClaimWorkers); i++ {
		go uc.queryPersistentVolumeClaim()
	}
	for i := 0; i < int(config.Config.Load.QueryVolumeAttachmentWorkers); i++ {
		go uc.queryVolumeAttachment()
	}
	for i := 0; i < int(config.Config.Load.QueryNodeWorkers); i++ {
		go uc.queryNode()
	}
	for i := 0; i < int(config.Config.Load.UpdateNodeWorkers); i++ {
		go uc.updateNode()
	}
	for i := 0; i < int(config.Config.Load.DeletePodWorkers); i++ {
		go uc.deletePod()
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
		klog.V(5).Infof("dispatch message content: %++v", msg)

		resourceType, err := messagelayer.GetResourceType(msg)
		if err != nil {
			klog.Warningf("parse message: %s resource type with error: %s", msg.GetID(), err)
			continue
		}

		operationType := msg.GetOperation()
		klog.V(5).Infof("message: %s, operation type is: %s", msg.GetID(), operationType)

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
			switch msg.GetOperation() {
			case model.QueryOperation:
				uc.queryNodeChan <- msg
			case model.UpdateOperation:
				uc.updateNodeChan <- msg
			default:
				klog.Errorf("message: %s, operation type: %s unsupported", msg.GetID(), msg.GetOperation())
			}
		case model.ResourceTypePod:
			if msg.GetOperation() == model.DeleteOperation {
				uc.podDeleteChan <- msg
			} else {
				klog.Errorf("message: %s, operation type: %s unsupported", msg.GetID(), msg.GetOperation())
			}
		default:
			klog.Errorf("message: %s, resource type: %s unsupported", msg.GetID(), resourceType)
		}
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

					if updatedPod, err := uc.kubeClient.CoreV1().Pods(getPod.Namespace).UpdateStatus(context.Background(), getPod, metaV1.UpdateOptions{}); err != nil {
						klog.Warningf("message: %s, update pod status failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, getPod.Namespace, getPod.Name)
					} else {
						klog.V(5).Infof("message: %s, update pod status successfully, namespace: %s, name: %s", msg.GetID(), updatedPod.Namespace, updatedPod.Name)
						if updatedPod.DeletionTimestamp != nil && (status.Phase == v1.PodSucceeded || status.Phase == v1.PodFailed) {
							if uc.isPodNotRunning(status.ContainerStatuses) {
								if err := uc.kubeClient.CoreV1().Pods(updatedPod.Namespace).Delete(context.Background(), updatedPod.Name, *metaV1.NewDeleteOptions(0)); err != nil {
									klog.Warningf("message: %s, graceful delete pod failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, updatedPod.Namespace, updatedPod.Name)
								}
								klog.Infof("message: %s, pod delete successfully, namespace: %s, name: %s", msg.GetID(), updatedPod.Namespace, updatedPod.Name)
							}
						}
					}
				}

			default:
				klog.Warningf("pod status operation: %s unsupported", msg.GetOperation())
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

func (uc *UpstreamController) updateNodeStatus() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop updateNodeStatus")
			return
		case msg := <-uc.nodeStatusChan:
			klog.V(5).Infof("message: %s, operation is: %s, and resource is %s", msg.GetID(), msg.GetOperation(), msg.GetResource())

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
					uc.nodeMsgResponse(name, namespace, "OK", msg)
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

				uc.nodeMsgResponse(name, namespace, "OK", msg)

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
				if !config.Config.EdgeSiteEnable {
					for i := range nodeStatusRequest.Status.Conditions {
						if time.Since(nodeStatusRequest.Status.Conditions[i].LastHeartbeatTime.Time) > time.Duration(config.Config.NodeUpdateFrequency)*time.Second {
							nodeStatusRequest.Status.Conditions[i].LastHeartbeatTime = metaV1.NewTime(time.Now())
						}

						if time.Since(nodeStatusRequest.Status.Conditions[i].LastTransitionTime.Time) > time.Duration(config.Config.NodeUpdateFrequency)*time.Second {
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

				// Keep the same "VolumesAttached" attribute with upstream,
				// since this value is maintained by kube-controller-manager.
				nodeStatusRequest.Status.VolumesAttached = getNode.Status.VolumesAttached

				getNode.Status = nodeStatusRequest.Status
				node, err := uc.kubeClient.CoreV1().Nodes().UpdateStatus(context.Background(), getNode, metaV1.UpdateOptions{})
				if err != nil {
					klog.Warningf("message: %s process failure, update node failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, getNode.Namespace, getNode.Name)
					continue
				}

				resMsg := model.NewMessage(msg.GetID())
				resMsg.SetResourceVersion(node.ResourceVersion)
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
				resMsg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				if err = uc.messageLayer.Response(*resMsg); err != nil {
					klog.Warningf("Message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}

				klog.V(4).Infof("message: %s, update node status successfully, namespace: %s, name: %s", msg.GetID(), getNode.Namespace, getNode.Name)

			default:
				klog.Warningf("message: %s process failure, node status operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
			klog.V(4).Infof("message: %s process successfully", msg.GetID())
		}
	}
}

func kubeClientGet(uc *UpstreamController, namespace string, name string, queryType string) (interface{}, string, error) {
	switch queryType {
	case model.ResourceTypeConfigmap:
		configMap, err := uc.kubeClient.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, metaV1.GetOptions{})
		resourceVersion := configMap.ResourceVersion
		return configMap, resourceVersion, err
	case model.ResourceTypeSecret:
		secret, err := uc.kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), name, metaV1.GetOptions{})
		resourceVersion := secret.ResourceVersion
		return secret, resourceVersion, err
	case common.ResourceTypeService:
		svc, err := uc.kubeClient.CoreV1().Services(namespace).Get(context.Background(), name, metaV1.GetOptions{})
		resourceVersion := svc.ResourceVersion
		return svc, resourceVersion, err
	case common.ResourceTypeEndpoints:
		eps, err := uc.kubeClient.CoreV1().Endpoints(namespace).Get(context.Background(), name, metaV1.GetOptions{})
		resourceVersion := eps.ResourceVersion
		return eps, resourceVersion, err
	case common.ResourceTypePersistentVolume:
		pv, err := uc.kubeClient.CoreV1().PersistentVolumes().Get(context.Background(), name, metaV1.GetOptions{})
		resourceVersion := pv.ResourceVersion
		return pv, resourceVersion, err
	case common.ResourceTypePersistentVolumeClaim:
		pvc, err := uc.kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metaV1.GetOptions{})
		resourceVersion := pvc.ResourceVersion
		return pvc, resourceVersion, err
	case common.ResourceTypeVolumeAttachment:
		va, err := uc.kubeClient.StorageV1().VolumeAttachments().Get(context.Background(), name, metaV1.GetOptions{})
		resourceVersion := va.ResourceVersion
		return va, resourceVersion, err
	case model.ResourceTypeNode:
		node, err := uc.kubeClient.CoreV1().Nodes().Get(context.Background(), name, metaV1.GetOptions{})
		resourceVersion := node.ResourceVersion
		return node, resourceVersion, err
	default:
		err := stderrors.New("Wrong query type")
		klog.Error(err)
		return nil, "", err
	}
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
		object, resourceVersion, err := kubeClientGet(uc, namespace, name, queryType)
		if errors.IsNotFound(err) {
			klog.Warningf("message: %s process failure, resource not found, namespace: %s, name: %s", msg.GetID(), namespace, name)
			return
		}
		if err != nil {
			klog.Warningf("message: %s process failure with error: %s, namespace: %s, name: %s", msg.GetID(), err, namespace, name)
			return
		}
		resMsg := model.NewMessage(msg.GetID())
		resMsg.SetResourceVersion(resourceVersion)
		resMsg.Content = object
		nodeID, err := messagelayer.GetNodeID(msg)
		if err != nil {
			klog.Warningf("message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
		}
		resource, err := messagelayer.BuildResource(nodeID, namespace, queryType, name)
		if err != nil {
			klog.Warningf("message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
		}
		resMsg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
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

func (uc *UpstreamController) queryService() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop queryService")
			return
		case msg := <-uc.serviceChan:
			queryInner(uc, msg, common.ResourceTypeService)
		}
	}
}

func (uc *UpstreamController) queryEndpoints() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop queryEndpoints")
			return
		case msg := <-uc.endpointsChan:
			queryInner(uc, msg, common.ResourceTypeEndpoints)
		}
	}
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

func (uc *UpstreamController) updateNode() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("stop updateNode")
			return
		case msg := <-uc.updateNodeChan:
			klog.V(5).Infof("message: %s, operation is: %s, and resource is %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
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
				getNode, err := uc.kubeClient.CoreV1().Nodes().Get(context.Background(), name, metaV1.GetOptions{})
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
				node, err := uc.kubeClient.CoreV1().Nodes().Update(context.Background(), getNode, metaV1.UpdateOptions{})
				if err != nil {
					klog.Warningf("message: %s process failure, update node failed with error: %s, namespace: %s, name: %s", msg.GetID(), err, getNode.Namespace, getNode.Name)
					continue
				}

				resMsg := model.NewMessage(msg.GetID())
				resMsg.SetResourceVersion(node.ResourceVersion)
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
				resMsg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
				if err = uc.messageLayer.Response(*resMsg); err != nil {
					klog.Warningf("Message: %s process failure, response failed with error: %s", msg.GetID(), err)
					continue
				}

				klog.V(4).Infof("message: %s, update node successfully, namespace: %s, name: %s", msg.GetID(), getNode.Namespace, getNode.Name)
			default:
				klog.Warningf("message: %s process failure, node operation: %s unsupported", msg.GetID(), msg.GetOperation())
			}
			klog.V(4).Infof("message: %s process successfully", msg.GetID())
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

// nodeMsgResponse response message of ResourceTypeNode
func (uc *UpstreamController) nodeMsgResponse(nodeName, namespace, content string, msg model.Message) {
	resMsg := model.NewMessage(msg.GetID())
	resMsg.Content = content
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

	resMsg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.ResponseOperation)
	if err = uc.messageLayer.Response(*resMsg); err != nil {
		klog.Warningf("Response message: %s failed, response failed with error: %s", msg.GetID(), err)
		return
	}
}

// NewUpstreamController create UpstreamController from config
func NewUpstreamController() (*UpstreamController, error) {
	cli, err := utils.KubeClient()
	if err != nil {
		klog.Warningf("create kube client failed with error: %s", err)
		return nil, err
	}
	uc := &UpstreamController{
		kubeClient:   cli,
		messageLayer: messagelayer.NewContextMessageLayer(),
	}
	return uc, nil
}
