package constants

// Service level constants
const (
	// module
	EdgeControllerModuleName     = "controller"
	CloudHubControllerModuleName = "cloudhub"

	// Resource sep
	ResourceSep               = "/"
	ResourceSliceLength        = 5
	ResourceSliceLengthQuery   = 4
	ResourceNodeIndex         = 0
	ResourceNodeIDIndex       = 1
	ResourceNamespaceIndex    = 2
	ResourceResourceTypeIndex = 3
	ResourceResourceNameIndex = 4
	ResourceNode              = "node"

	// Group
	GroupResource = "resource"

	// Nvidia Constants
	// NvidiaGPUStatusAnnotationKey is the key of the node annotation for GPU status
	NvidiaGPUStatusAnnotationKey = "huawei.com/gpu-status"
	// NvidiaGPUDecisionAnnotationKey is the key of the pod annotation for scheduler GPU decision
	NvidiaGPUDecisionAnnotationKey = "huawei.com/gpu-decision"
	// NvidiaGPUScalarResourceName is the device plugin resource name used for special handling
	NvidiaGPUScalarResourceName = "nvidia.com/gpu"
	// NvidiaGPUMaxUsage is the maximum possible usage of a GPU in millis
	NvidiaGPUMaxUsage = 1000
)
