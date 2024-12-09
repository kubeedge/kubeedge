package driver

import (
	"sync"

	"github.com/kubeedge/mapper-framework/pkg/common"
)

func NewClient(protocol ProtocolConfig) (*CustomizedClient, error) {
	client := &CustomizedClient{
		ProtocolConfig: protocol,
		deviceMutex:    sync.Mutex{},
		// TODO initialize the variables you added
	}
	return client, nil
}

func (c *CustomizedClient) InitDevice() error {
	// TODO: add init operation
	// you can use c.ProtocolConfig
	return nil
}

func (c *CustomizedClient) GetDeviceData(visitor *VisitorConfig) (interface{}, error) {
	// TODO: add the code to get device's data
	// you can use c.ProtocolConfig and visitor
	return nil, nil
}

func (c *CustomizedClient) DeviceDataWrite(visitor *VisitorConfig, deviceMethodName string, propertyName string, data interface{}) error {
	// TODO: add the code to write device's data
	// you can use c.ProtocolConfig and visitor to write data to device
	return nil
}

func (c *CustomizedClient) SetDeviceData(data interface{}, visitor *VisitorConfig) error {
	// TODO: set device's data
	// you can use c.ProtocolConfig and visitor
	return nil
}

func (c *CustomizedClient) StopDevice() error {
	// TODO: stop device
	// you can use c.ProtocolConfig
	return nil
}

func (c *CustomizedClient) GetDeviceStates() (string, error) {
	// TODO: GetDeviceStates
	return common.DeviceStatusOK, nil
}
