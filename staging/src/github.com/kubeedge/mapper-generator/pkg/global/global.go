package global

import (
	"github.com/kubeedge/mapper-generator/pkg/common"
	"github.com/kubeedge/mapper-generator/pkg/config"
)

// DevPanel defined operations on devices, manage the lifecycle of devices
type DevPanel interface {
	// DevStart start device to collect/push/save data to edgecore/app/database
	DevStart()
	// DevInit get device info by dmi interface
	DevInit(cfg *config.Config) error
	// UpdateDev update device's config and restart the device
	UpdateDev(model *common.DeviceModel, device *common.DeviceInstance, protocol *common.Protocol)
	// UpdateDevTwins update device twin's config and restart the device
	UpdateDevTwins(deviceID string, twins []common.Twin) error
	// DealDeviceTwinGet get device's twin data
	DealDeviceTwinGet(deviceID string, twinName string) (interface{}, error)
	// GetDevice get device's instance info
	GetDevice(deviceID string) (interface{}, error)
	// RemoveDevice stop device and remove device
	RemoveDevice(deviceID string) error
	// GetModel get model's info
	GetModel(modelName string) (common.DeviceModel, error)
	// UpdateModel update model in map only
	UpdateModel(model *common.DeviceModel)
	// RemoveModel remove model in map only
	RemoveModel(modelName string)
	// GetTwinResult get device's property value and datatype
	GetTwinResult(deviceID string, twinName string) (string, string, error)
}

// DataPanel defined push method, parse the push operation in CRD and execute it
type DataPanel interface {
	// TODO add more interface

	// InitPushMethod initialization operation before push
	InitPushMethod() error
	// Push implement push operation
	Push(data *common.DataModel)
}

// DataBaseClient defined database interface, save data and provide data to REST API
type DataBaseClient interface {
	// TODO add more interface

	InitDbClient() error
	CloseSession()

	AddData(data *common.DataModel)

	GetDataByDeviceName(deviceName string) ([]*common.DataModel, error)
	GetPropertyDataByDeviceName(deviceName string, propertyData string) ([]*common.DataModel, error)
	GetDataByTimeRange(start int64, end int64) ([]*common.DataModel, error)

	DeleteDataByTimeRange(start int64, end int64) ([]*common.DataModel, error)
}
