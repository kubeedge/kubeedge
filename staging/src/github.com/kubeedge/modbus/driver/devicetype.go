package driver

import (
	"sync"

	"github.com/kubeedge/mapper-framework/pkg/common"
)

// CustomizedDev is the customized device configuration and client information.
type CustomizedDev struct {
	Instance         common.DeviceInstance
	CustomizedClient *CustomizedClient
}

type CustomizedClient struct {
	// TODO add some variables to help you better implement device drivers
	intMaxValue int
	deviceMutex sync.Mutex
	ProtocolConfig
}

type ProtocolConfig struct {
	ProtocolName string `json:"protocolName"`
	ConfigData   `json:"configData"`
}

type ConfigData struct {
	// TODO: add your protocol config data
	DeviceID   string `json:"DeviceID,omitempty"`
	SerialPort string `json:"SerialPort"`
	DataBits   string `json:"DataBits"`
	BaudRate   string `json:"BaudRate"`
	Parity     string `json:"Parity"`
	StopBits   string `json:"StopBits"`
	ProtocolID string `json:"ProtocolID"`
}

type VisitorConfig struct {
	ProtocolName      string `json:"protocolName"`
	VisitorConfigData `json:"configData"`
}

type VisitorConfigData struct {
	// TODO: add your visitor config data
	DataType       string `json:"DataType"`
	Register       string `json:"Register"`
	Offset         string `json:"Offset"`
	Limit          string `json:"Limit"`
	Scale          string `json:"Scale"`
	IsSwap         string `json:"IsSwap"`
	IsRegisterSwap string `json:"IsRegisterSwap"`
}
