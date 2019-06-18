package constants

import (
	"k8s.io/api/core/v1"
)

// Config
const (
	DefaultKubeContentType = "application/vnd.kubernetes.protobuf"
	DefaultKubeNamespace   = v1.NamespaceAll
	DefaultKubeQPS         = 100.0
	DefaultKubeBurst       = 10

	DefaultUpdateDeviceStatusWorkers = 1

	DefaultUpdateDeviceStatusBuffer = 1024

	DefaultMessageLayer = "context"

	DefaultContextSendModuleName     = CloudHubControllerModuleName
	DefaultContextReceiveModuleName  = DeviceControllerModuleName
	DefaultContextResponseModuleName = CloudHubControllerModuleName

	DefaultDeviceEventBuffer      = 1
	DefaultDeviceModelEventBuffer = 1
)
