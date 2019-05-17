package constants

// Service level constants
const (
	// module
	EdgeControllerModuleName     = "controller"
	CloudHubControllerModuleName = "cloudhub"

	// Resource sep
	ResourceSep               = "/"
	ResourceSliceLength       = 5
	ResourceSliceLengthQuery  = 4
	ResourceNodeIndex         = 0
        ResourceNodeIDIndex       = 1
	ResourceNamespaceIndex    = 0
	ResourceResourceTypeIndex = 1
	ResourceResourceNameIndex = 2
	ResourceNode              = "node"

	// Group
	GroupResource = "resource"

	// Nvidia Constants
	// NvidiaGPUStatusAnnotationKey is the key of the node annotation for GPU status
	NvidiaGPUStatusAnnotationKey = "gpu.kubeedge.io/gpu-status"
	// NvidiaGPUDecisionAnnotationKey is the key of the pod annotation for scheduler GPU decision
	NvidiaGPUDecisionAnnotationKey = "gpu.kubeedge.io/gpu-decision"
	// NvidiaGPUScalarResourceName is the device plugin resource name used for special handling
	NvidiaGPUScalarResourceName = "gpu.kubeedge.io/gpu"
	// NvidiaGPUMaxUsage is the maximum possible usage of a GPU in millis
	NvidiaGPUMaxUsage = 1000
)
