/*
Copyright 2019 The KubeEdge Authors.

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
package apis

import (
	"errors"
	"time"
)

const (
	// NvidiaGPUStatusAnnotationKey is the key of the node annotation for GPU status
	NvidiaGPUStatusAnnotationKey = "huawei.com/gpu-status"
	// NvidiaGPUDecisionAnnotationKey is the key of the pod annotation for scheduler GPU decision
	NvidiaGPUDecisionAnnotationKey = "huawei.com/gpu-decision"
	// NvidiaGPUScalarResourceName is the device plugin resource name used for special handling
	NvidiaGPUScalarResourceName = "nvidia.com/gpu"
	// NvidiaGPUMaxUsage is the maximum possible usage of a GPU in millis
	NvidiaGPUMaxUsage = 1000
	//StatusTag is to compare status of resources
	StatusTag = "StatusTag"
)

//RainerCoreConfiguration is configuration object for edge node
type RainerCoreConfiguration struct {

	// IP address for the EdgeStore etcd server
	EdgeStoreAddress string

	// Port for the EdgeStore to server on
	EdgeStorePort int

	// nodesStatusUpdateFrequency is the frequency that EdgeController posts edgenodes
	// status to apiserver. Note: be cautious when changing the constant, it
	// must work with nodeMonitorGracePeriod in nodecontroller.
	NodesStatusUpdateFrequency time.Duration

	// Todo: for now, assume single namespace
	// registerNodeNamespace defines the namespace of the edge nodes to be registered
	RegisterNodeNamespace string
}

//Node is object type for node
type Node struct {
	Name string
}

//UID is string form to represent ID
type UID string

//error variables
var (
	ErrPodNotFound       = errors.New("PodNotFound")
	ErrContainerNotFound = errors.New("ContainerNotFound")
	ErrPodStartBackOff   = errors.New("PodStartBackOff")
)
