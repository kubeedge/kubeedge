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

	StatusTag = "StatusTag"
)

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

type Node struct {
	Name string
}

type UID string

var (
	ErrPodNotFound       = errors.New("PodNotFound")
	ErrContainerNotFound = errors.New("ContainerNotFound")
	ErrPodStartBackOff   = errors.New("PodStartBackOff")
)
