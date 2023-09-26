package driver

import (
	"sync"

	"github.com/kubeedge/mapper-generator/pkg/common"
)

// CustomizedDev is the customized device configuration and client information.
type CustomizedDev struct {
	Instance         common.DeviceInstance
	CustomizedClient *CustomizedClient
}

type CustomizedClient struct {
	// TODO add some variables to help you better implement device drivers
	deviceMutex sync.Mutex
	ProtocolConfig
}

type ProtocolConfig struct { //customizedprotocol字段
	ProtocolName string `json:"protocolName"`
	ConfigData   `json:"configData"`
}

type ConfigData struct {
	// TODO: add your config data according to configmap
}

type VisitorConfig struct {
	ProtocolName      string `json:"protocolName"`
	VisitorConfigData `json:"configData"`
}

type VisitorConfigData struct {
	// TODO: add your Visitor ConfigData according to configmap
	DataType string `json:"dataType"`
}
