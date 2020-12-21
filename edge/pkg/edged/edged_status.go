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
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/apis"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	"github.com/kubeedge/kubeedge/pkg/util"
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
		klog.Errorf("couldn't determine hostname: %v", err)
		return nil, err
	}
	if len(e.nodeName) != 0 {
		hostname = e.nodeName
	}

	node.Labels = map[string]string{
		// Kubernetes built-in labels
		v1.LabelHostname:   hostname,
		v1.LabelOSStable:   runtime.GOOS,
		v1.LabelArchStable: runtime.GOARCH,

		// KubeEdge specific labels
		"node-role.kubernetes.io/edge":  "",
		"node-role.kubernetes.io/agent": "",
	}

	node.Status.Addresses = []v1.NodeAddress{
		{Type: v1.NodeInternalIP, Address: e.nodeIP.String()},
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
	klog.Infof("retrieve piggybacked status: %v", statusList)
	return statusList, nil
}

func (e *edged) getNodeStatusRequest(node *v1.Node) (*edgeapi.NodeStatusRequest, error) {
	var nodeStatus = &edgeapi.NodeStatusRequest{}
	nodeStatus.UID = e.uid
	nodeStatus.Status = *node.Status.DeepCopy()
	nodeStatus.Status.Phase = e.getNodePhase()

	devicePluginCapacity, _, removedDevicePlugins := e.getDevicePluginResourceCapacity()
	for k, v := range devicePluginCapacity {
		klog.Infof("Update capacity for %s to %d", k, v.Value())
		nodeStatus.Status.Capacity[k] = v
		nodeStatus.Status.Allocatable[k] = v
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
				klog.Infof("Setting node annotation to add node status list to Scheduler")
				continue
			}
		}
		klog.Infof("Remove capacity for %s", removedResource)
		delete(node.Status.Capacity, v1.ResourceName(removedResource))
	}
	e.setNodeStatusDaemonEndpoints(nodeStatus)
	e.setNodeStatusConditions(nodeStatus)
	if e.gpuPluginEnabled {
		err := e.setGPUInfo(nodeStatus)
		if err != nil {
			klog.Errorf("setGPUInfo failed, err: %v", err)
		}
	}
	if e.volumeManager.ReconcilerStatesHasBeenSynced() {
		node.Status.VolumesInUse = e.volumeManager.GetVolumesInUse()
	} else {
		node.Status.VolumesInUse = nil
	}
	e.volumeManager.MarkVolumesAsReportedInUse(node.Status.VolumesInUse)
	klog.Infof("Sync VolumesInUse: %v", node.Status.VolumesInUse)

	return nodeStatus, nil
}

func (e *edged) setNodeStatusDaemonEndpoints(node *edgeapi.NodeStatusRequest) {
	node.Status.DaemonEndpoints = v1.NodeDaemonEndpoints{
		KubeletEndpoint: v1.DaemonEndpoint{
			Port: constants.ServerPort,
		},
	}
}

func (e *edged) setNodeStatusConditions(node *edgeapi.NodeStatusRequest) {
	e.setNodeReadyCondition(node)
}

// setNodeReadyCondition is partially come from "k8s.io/kubernetes/pkg/kubelet.setNodeReadyCondition"
func (e *edged) setNodeReadyCondition(node *edgeapi.NodeStatusRequest) {
	currentTime := metav1.NewTime(time.Now())
	var newNodeReadyCondition v1.NodeCondition

	var err error
	_, err = e.containerRuntime.Version()

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

	runtimeVersion, err := e.containerRuntime.Version()
	if err != nil {
		return nodeInfo, err
	}
	nodeInfo.ContainerRuntimeVersion = fmt.Sprintf("%s://%s", e.containerRuntimeName, runtimeVersion.String())

	nodeInfo.KernelVersion = kernel
	nodeInfo.OperatingSystem = runtime.GOOS
	nodeInfo.Architecture = runtime.GOARCH
	nodeInfo.KubeletVersion = e.version
	nodeInfo.OSImage = prettyName

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
			klog.Errorf("parse gpu failed, gpuInfo: %v, params: %v", gpuInfo, params)
			continue
		}
		gpuName := params[1]
		gpuType := params[2]
		result, err = util.Command("sh", []string{"-c", fmt.Sprintf("%s -i %s -a|grep -A 3 \"FB Memory Usage\"| grep Total", GPUInfoQueryTool, gpuName)})
		if err != nil {
			klog.Errorf("get gpu(%v) memory failed, err: %v", gpuName, err)
			continue
		}
		parts := strings.Split(result, ":")
		if len(parts) != 2 {
			klog.Errorf("parse gpu(%v) memory failed, parts: %v", gpuName, parts)
			continue
		}
		mem := strings.TrimSpace(strings.Split(strings.TrimSpace(parts[1]), " ")[0])

		gpuResource := edgeapi.ExtendResource{}
		gpuResource.Name = fmt.Sprintf("nvidia%v", gpuName)
		gpuResource.Type = gpuType
		gpuResource.Capacity = resource.MustParse(mem + "Mi")
		gpuResources = append(gpuResources, gpuResource)
	}

	nodeStatus.ExtendResources[apis.NvidiaGPUResource] = gpuResources
	return nil
}

func (e *edged) setMemInfo(total, allocated v1.ResourceList) error {
	out, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return err
	}
	matches := regexp.MustCompile(`MemTotal:\s*([0-9]+) kB`).FindSubmatch(out)
	if len(matches) != 2 {
		return fmt.Errorf("failed to match regexp in output: %q", string(out))
	}
	m, err := strconv.ParseInt(string(matches[1]), 10, 64)
	if err != nil {
		return err
	}
	totalMem := m / 1024
	mem := resource.MustParse(strconv.FormatInt(totalMem, 10) + "Mi")
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

func (e *edged) getDevicePluginResourceCapacity() (v1.ResourceList, v1.ResourceList, []string) {
	return e.containerManager.GetDevicePluginResourceCapacity()
}

func (e *edged) getNodePhase() v1.NodePhase {
	return v1.NodeRunning
}

func (e *edged) registerNode() error {
	node, err := e.initialNode()
	if err != nil {
		klog.Errorf("Unable to construct v1.Node object for edge: %v", err)
		return err
	}

	e.setInitNode(node)

	if !config.Config.RegisterNode {
		//when register-node set to false, do not auto register node
		klog.Infof("register-node is set to false")
		e.registrationCompleted = true
		return nil
	}

	klog.Infof("Attempting to register node %s", e.nodeName)

	resource := fmt.Sprintf("%s/%s/%s", e.namespace, model.ResourceTypeNodeStatus, e.nodeName)
	nodeInfoMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, node)
	var res model.Message
	if _, ok := core.GetModules()[edgehub.ModuleNameEdgeHub]; ok {
		res, err = beehiveContext.SendSync(edgehub.ModuleNameEdgeHub, *nodeInfoMsg, syncMsgRespTimeout)
	} else {
		res, err = beehiveContext.SendSync(EdgeController, *nodeInfoMsg, syncMsgRespTimeout)
	}

	if err != nil || res.Content != "OK" {
		klog.Errorf("register node failed, error: %v", err)
		if res.Content != "OK" {
			klog.Errorf("response from cloud core: %v", res.Content)
		}
		return err
	}

	klog.Infof("Successfully registered node %s", e.nodeName)
	e.registrationCompleted = true

	return nil
}

func (e *edged) updateNodeStatus() error {
	nodeStatus, err := e.getNodeStatusRequest(&initNode)
	if err != nil {
		klog.Errorf("Unable to construct api.NodeStatusRequest object for edge: %v", err)
		return err
	}

	err = e.metaClient.NodeStatus(e.namespace).Update(e.nodeName, *nodeStatus)
	if err != nil {
		klog.Errorf("update node failed, error: %v", err)
	}
	return nil
}

func (e *edged) syncNodeStatus() {
	if !e.registrationCompleted {
		if err := e.registerNode(); err != nil {
			klog.Errorf("Register node failed: %v", err)
			return
		}
	}

	if err := e.updateNodeStatus(); err != nil {
		klog.Errorf("Unable to update node status: %v", err)
	}
}
