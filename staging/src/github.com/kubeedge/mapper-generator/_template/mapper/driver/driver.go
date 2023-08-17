package driver

import (
	"sync"
)

func NewClient(commonProtocol ProtocolCommonConfig,
	protocol ProtocolConfig) (*CustomizedClient, error) {
	client := &CustomizedClient{
		ProtocolCommonConfig: commonProtocol,
		ProtocolConfig:       protocol,
		deviceMutex:          sync.Mutex{},
		// TODO initialize the variables you added
	}
	return client, nil
}

func (c *CustomizedClient) InitDevice() error {
	// TODO: add init operation
	// you can use c.ProtocolConfig and c.ProtocolCommonConfig
	return nil
}

func (c *CustomizedClient) GetDeviceData(visitor *VisitorConfig) (interface{}, error) {
	// TODO: get device's data
	// you can use c.ProtocolConfig,c.ProtocolCommonConfig and visitor
	return nil, nil
}

func (c *CustomizedClient) SetDeviceData(data interface{}, visitor *VisitorConfig) error {
	// TODO: set device's data
	// you can use c.ProtocolConfig,c.ProtocolCommonConfig and visitor
	return nil
}

func (c *CustomizedClient) StopDevice() error {
	// TODO: stop device
	// you can use c.ProtocolConfig and c.ProtocolCommonConfig
	return nil
}
