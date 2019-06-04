package constants

// Service level constants
const (
	// module
	EdgeControllerModuleName = "controller"

	ResourceNodeIDIndex       = 1
	ResourceNamespaceIndex    = 2
	ResourceResourceTypeIndex = 3
	ResourceResourceNameIndex = 4

	EdgeSiteResourceNamespaceIndex    = 0
	EdgeSiteResourceResourceTypeIndex = 1
	EdgeSiteResourceResourceNameIndex = 2

	ResourceNode = "node"

	// Group
	GroupResource = "resource"

	// Nvidia Constants
	// NvidiaGPUStatusAnnotationKey is the key of the node annotation for GPU status
	NvidiaGPUStatusAnnotationKey = "huawei.com/gpu-status"
	// NvidiaGPUScalarResourceName is the device plugin resource name used for special handling
	NvidiaGPUScalarResourceName = "nvidia.com/gpu"
)
