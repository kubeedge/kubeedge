package driver

import "encoding/json"

// Adapter is the interface that users must implement for their specific protocol.
type Adapter interface {
	// InitDevice initializes the device.
	InitDevice() error
	// GetDeviceData reads data from the device based on the visitor configuration.
	GetDeviceData(visitor *VisitorConfig) (interface{}, error)
	// DeviceDataWrite writes data to the device.
	DeviceDataWrite(visitor *VisitorConfig, deviceMethodName string, propertyName string, data interface{}) error
	// SetDeviceData sets device's data.
	SetDeviceData(data interface{}, visitor *VisitorConfig) error
	// StopDevice stops the device.
	StopDevice() error
	// GetDeviceStates returns the state of the device.
	GetDeviceStates() (string, error)
	// AnomalyDetectionProcess handles anomaly detection.
	AnomalyDetectionProcess(req *AnomalyDetectionRequest) error
}

// VisitorConfig defines the configuration required to interact with a specific device property.
type VisitorConfig struct {
	ProtocolName      string          `json:"protocolName"`
	VisitorConfigData json.RawMessage `json:"configData"`
}

// AnomalyDetectionRequest defines the anomaly detection payload.
type AnomalyDetectionRequest struct {
	Enabled                bool            `json:"enabled"`
	VisitorConfig          VisitorConfig   `json:"visitorConfig"`
	AnomalyDetectionConfig json.RawMessage `json:"anomalyDetectionConfig"`
	Data                   interface{}     `json:"data"`
}
