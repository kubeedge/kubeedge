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
1. setNodeReadyCondition is partially come from "k8s.io/kubernetes/pkg/kubelet.setNodeReadyCondition"
*/

package edged

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/apis"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/util"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

//GPUInfoQueryTool sets information monitoring tool location for GPU
var GPUInfoQueryTool = "/var/IEF/nvidia/bin/nvidia-smi"
var initNode v1.Node
var reservationMemory = resource.MustParse(fmt.Sprintf("%dMi", 100))

func (e *edged) initialNode() (*v1.Node, error) {
	var node = &v1.Node{}

	if runtime.GOOS == "windows" {
		return node, nil
	}

	nodeInfo, err := e.getNodeInfo()
	if err != nil {
		return nil, err
	}
	node.Status.NodeInfo = nodeInfo

	hostname, err := os.Hostname()
	if err != nil {
		log.LOGGER.Errorf("couldn't determine hostname: %v", err)
	}

	ip, err := e.getIP()
	if err != nil {
		return nil, err
	}
	node.Status.Addresses = []v1.NodeAddress{
		{Type: v1.NodeInternalIP, Address: ip},
		{Type: v1.NodeHostName, Address: hostname},
	}

	node.Status.Capacity = make(v1.ResourceList)
	node.Status.Allocatable = make(v1.ResourceList)
	err = e.setMemInfo(node.Status.Capacity, node.Status.Allocatable)
	if err != nil {
		return nil, err
	}

	err = e.setCPUInfo(node.Status.Capacity, node.Status.Allocatable)
	if err != nil {
		return nil, err
	}

	node.Status.Capacity[v1.ResourcePods] = *resource.NewQuantity(110, resource.DecimalSI)
	node.Status.Allocatable[v1.ResourcePods] = *resource.NewQuantity(110, resource.DecimalSI)

	return node, nil
}

func (e *edged) setInitNode(node *v1.Node) {
	initNode.Status = *node.Status.DeepCopy()
}

// Retrieve node status
func retrieveDevicePluginStatus(s string) (string, error) {
	tagLen := len(apis.StatusTag)
	if len(s) <= tagLen {
		return "", fmt.Errorf("no node status wrapped in")
	}

	tag := s[:tagLen]
	if string(tag) != apis.StatusTag {
		return "", fmt.Errorf("not a node status json string")
	}
	statusList := s[tagLen:]
	log.LOGGER.Infof("retrieve piggybacked status: %v", statusList)
	return statusList, nil
}

func (e *edged) getNodeStatusRequest(node *v1.Node) (*edgeapi.NodeStatusRequest, error) {
	var nodeStatus = &edgeapi.NodeStatusRequest{}
	nodeStatus.UID = e.uid
	nodeStatus.Status = *node.Status.DeepCopy()
	nodeStatus.Status.Phase = e.getNodePhase()

	devicePluginCapacity, removedDevicePlugins := e.getDevicePluginResourceCapacity()
	if devicePluginCapacity != nil {
		for k, v := range devicePluginCapacity {
			log.LOGGER.Infof("Update capacity for %s to %d", k, v.Value())
			nodeStatus.Status.Capacity[k] = v
			nodeStatus.Status.Allocatable[k] = v
		}
	}

	nameSet := sets.NewString(string(v1.ResourceCPU), string(v1.ResourceMemory), string(v1.ResourceStorage),
		string(v1.ResourceEphemeralStorage), string(apis.NvidiaGPUScalarResourceName))

	for _, removedResource := range removedDevicePlugins {
		// if the remmovedReousrce is not contained in the nameSet and contains specific tag
		if !nameSet.Has(removedResource) {
			status, err := retrieveDevicePluginStatus(removedResource)
			if err == nil {
				if node.Annotations == nil {
					node.Annotations = make(map[string]string)
				}
				node.Annotations[apis.NvidiaGPUStatusAnnotationKey] = status
				log.LOGGER.Infof("Setting node annotation to add node status list to Scheduler")
				continue
			}
		}
		log.LOGGER.Infof("Remove capacity for %s", removedResource)
		delete(node.Status.Capacity, v1.ResourceName(removedResource))
	}

	e.setNodeStatusConditions(nodeStatus)
	if e.gpuPluginEnabled {
		err := e.setGPUInfo(nodeStatus)
		if err != nil {
			log.LOGGER.Errorf("setGPUInfo failed, err: %v", err)
		}
	}

	return nodeStatus, nil
}

func (e *edged) setNodeStatusConditions(node *edgeapi.NodeStatusRequest) {
	e.setNodeReadyCondition(node)
}

// setNodeReadyCondition is partially come from "k8s.io/kubernetes/pkg/kubelet.setNodeReadyCondition"
func (e *edged) setNodeReadyCondition(node *edgeapi.NodeStatusRequest) {
	currentTime := metav1.NewTime(time.Now())
	var newNodeReadyCondition v1.NodeCondition

	var err error
	switch e.containerRuntimeName {
	case DockerContainerRuntime:
		_, err = e.runtime.Version()
	case RemoteContainerRuntime:
		_, err = e.containerRuntime.Version()
	default:
	}

	if err != nil {
		newNodeReadyCondition = v1.NodeCondition{
			Type:              v1.NodeReady,
			Status:            v1.ConditionFalse,
			Reason:            "EdgeNotReady",
			Message:           err.Error(),
			LastHeartbeatTime: currentTime,
		}
	} else {
		newNodeReadyCondition = v1.NodeCondition{
			Type:              v1.NodeReady,
			Status:            v1.ConditionTrue,
			Reason:            "EdgeReady",
			Message:           "edge is posting ready status",
			LastHeartbeatTime: currentTime,
		}
	}

	readyConditionUpdated := false
	for i := range node.Status.Conditions {
		if node.Status.Conditions[i].Type == v1.NodeReady {
			if node.Status.Conditions[i].Status == newNodeReadyCondition.Status {
				newNodeReadyCondition.LastTransitionTime = node.Status.Conditions[i].LastTransitionTime
			} else {
				newNodeReadyCondition.LastTransitionTime = currentTime
			}
			node.Status.Conditions[i] = newNodeReadyCondition
			readyConditionUpdated = true
			break
		}
	}
	if !readyConditionUpdated {
		newNodeReadyCondition.LastTransitionTime = currentTime
		node.Status.Conditions = append(node.Status.Conditions, newNodeReadyCondition)
	}

}

func (e *edged) getNodeInfo() (v1.NodeSystemInfo, error) {
	nodeInfo := v1.NodeSystemInfo{}
	kernel, err := util.Command("uname", []string{"-r"})
	if err != nil {
		return nodeInfo, err
	}

	prettyName, err := util.Command("sh", []string{"-c", `cat /etc/os-release | grep PRETTY_NAME| awk -F '"' '{print$2}'`})
	if err != nil {
		return nodeInfo, err
	}

	switch e.containerRuntimeName {
	case DockerContainerRuntime:
		runtimeVersion, err := e.runtime.Version()
		if err != nil {
			return nodeInfo, err
		}
		nodeInfo.ContainerRuntimeVersion = fmt.Sprintf("docker://%s", runtimeVersion.String())
	case RemoteContainerRuntime:
		runtimeVersion, err := e.containerRuntime.Version()
		if err != nil {
			return nodeInfo, err
		}
		nodeInfo.ContainerRuntimeVersion = fmt.Sprintf("remote://%s", runtimeVersion.String())
	default:
	}

	nodeInfo.KernelVersion = kernel
	nodeInfo.OperatingSystem = runtime.GOOS
	nodeInfo.Architecture = runtime.GOARCH
	nodeInfo.KubeletVersion = e.version
	nodeInfo.OSImage = prettyName
	//nodeInfo.ContainerRuntimeVersion = fmt.Sprintf("docker://%s", runtimeVersion.String())

	return nodeInfo, nil

}

func (e *edged) setGPUInfo(nodeStatus *edgeapi.NodeStatusRequest) error {
	_, err := os.Stat(GPUInfoQueryTool)
	if err != nil {
		return fmt.Errorf("can not get file in path: %s, err: %v", GPUInfoQueryTool, err)
	}

	nodeStatus.ExtendResources = make(map[v1.ResourceName][]edgeapi.ExtendResource)

	result, err := util.Command("sh", []string{"-c", fmt.Sprintf("%s -L", GPUInfoQueryTool)})
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`GPU .*:.*\(.*\)`)
	gpuInfos := re.FindAllString(result, -1)
	gpuResources := make([]edgeapi.ExtendResource, 0)
	gpuRegexp := regexp.MustCompile(`^GPU ([\d]+):(.*)\(.*\)`)
	for _, gpuInfo := range gpuInfos {
		params := gpuRegexp.FindStringSubmatch(strings.TrimSpace(gpuInfo))
		if len(params) != 3 {
			log.LOGGER.Errorf("parse gpu failed, gpuInfo: %v, params: %v", gpuInfo, params)
			continue
		}
		gpuName := params[1]
		gpuType := params[2]
		result, err = util.Command("sh", []string{"-c", fmt.Sprintf("%s -i %s -a|grep -A 3 \"FB Memory Usage\"| grep Total", GPUInfoQueryTool, gpuName)})
		if err != nil {
			log.LOGGER.Errorf("get gpu(%v) memory failed, err: %v", gpuName, err)
			continue
		}
		parts := strings.Split(result, ":")
		if len(parts) != 2 {
			log.LOGGER.Errorf("parse gpu(%v) memory failed, parts: %v", gpuName, parts)
			continue
		}
		mem := strings.TrimSpace(strings.Split(strings.TrimSpace(parts[1]), " ")[0])

		gpuResource := edgeapi.ExtendResource{}
		gpuResource.Name = fmt.Sprintf("nvidia%v", gpuName)
		gpuResource.Type = gpuType
		gpuResource.Capacity = resource.MustParse(mem + "Mi")
		gpuResources = append(gpuResources, gpuResource)
	}

	nodeStatus.ExtendResources[v1.ResourceNvidiaGPU] = gpuResources
	return nil
}

func (e *edged) getIP() (string, error) {
	var ipAddr net.IP
	var err error
	addrs, _ := net.LookupIP(e.nodeName)
	for _, addr := range addrs {
		if err := util.ValidateNodeIP(addr); err == nil {
			if addr.To4() != nil {
				ipAddr = addr
				break
			}
			if addr.To16() != nil && ipAddr == nil {
				ipAddr = addr
			}
		}
	}

	if ipAddr == nil {
		ipAddr, err = util.ChooseHostInterface()
	}

	if err != nil {
		return "", err
	}

	return ipAddr.String(), nil
}

func (e *edged) setMemInfo(total, allocated v1.ResourceList) error {
	totalMem, err := util.Command("/bin/sh", []string{"-c", `free -m | grep Mem | awk '{print$2}'`})
	if err != nil {
		return err
	}
	mem := resource.MustParse(totalMem + "Mi")
	total[v1.ResourceMemory] = mem.DeepCopy()

	if mem.Cmp(reservationMemory) > 0 {
		mem.Sub(reservationMemory)
	}
	allocated[v1.ResourceMemory] = mem.DeepCopy()

	return nil
}

func (e *edged) setCPUInfo(total, allocated v1.ResourceList) error {
	total[v1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%d", runtime.NumCPU()))
	allocated[v1.ResourceCPU] = total[v1.ResourceCPU].DeepCopy()

	return nil
}

func (e *edged) getDevicePluginResourceCapacity() (v1.ResourceList, []string) {
	switch e.containerRuntimeName {
	case DockerContainerRuntime:
		return e.runtime.GetDevicePluginResourceCapacity()
	case RemoteContainerRuntime:
		//resourceList, _, str := e.containerManager.GetDevicePluginResourceCapacity()
		//return resourceList, str
	default:
	}
	return nil, nil
}

func (e *edged) getNodePhase() v1.NodePhase {
	return v1.NodeRunning
}

func (e *edged) registerNode() {
	step := 100 * time.Millisecond

	for {
		time.Sleep(step)
		step = step * 2
		if step >= 7*time.Second {
			step = 7 * time.Second
		}

		node, err := e.initialNode()
		if err != nil {
			log.LOGGER.Errorf("Unable to construct v1.Node object for edge: %v", err)
			continue
		}

		e.setInitNode(node)

		nodeStatus, err := e.getNodeStatusRequest(node)
		if err != nil {
			log.LOGGER.Errorf("Unable to construct api.NodeStatusRequest object for edge: %v", err)
			continue
		}

		log.LOGGER.Infof("Attempting to register node %s", e.nodeName)
		registered := e.tryRegisterToMeta(nodeStatus)
		if registered {
			log.LOGGER.Infof("Successfully registered node %s", e.nodeName)
			e.registrationCompleted = true
			return
		}
	}
}

func (e *edged) tryRegisterToMeta(node *edgeapi.NodeStatusRequest) bool {
	err := e.metaClient.NodeStatus(e.namespace).Update(e.nodeName, *node)
	if err != nil {
		log.LOGGER.Errorf("register node failed, error: %v", err)
	}
	return true
}

func (e *edged) updateNodeStatus() error {
	nodeStatus, err := e.getNodeStatusRequest(&initNode)
	if err != nil {
		log.LOGGER.Errorf("Unable to construct api.NodeStatusRequest object for edge: %v", err)
		return err
	}

	err = e.metaClient.NodeStatus(e.namespace).Update(e.nodeName, *nodeStatus)
	if err != nil {
		log.LOGGER.Errorf("update node failed, error: %v", err)
	}
	return nil
}

func (e *edged) syncNodeStatus() {
	if !e.registrationCompleted {
		// This will exit immediately if it doesn't need to do anything.
		e.registerNode()
	}
	if err := e.updateNodeStatus(); err != nil {
		log.LOGGER.Errorf("Unable to update node status: %v", err)
	}
}
