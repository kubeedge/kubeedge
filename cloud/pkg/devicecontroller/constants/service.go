package constants

import "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"

// Service level constants
const (
	// module
	DeviceControllerModuleName   = "devicecontroller"
	CloudHubControllerModuleName = "cloudhub"

	// group
	DeviceControllerModuleGroup = model.SrcDeviceController

	ResourceDeviceIndex         = 2
	ResourceDeviceIDIndex       = 3
	ResourceNode                = "node"
	ResourceDevice              = "device"
	ResourceTypeTwinEdgeUpdated = "twin/edge_updated"

	// Group
	GroupTwin = "twin"
)
