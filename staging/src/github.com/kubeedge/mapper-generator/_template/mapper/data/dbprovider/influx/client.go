package influx

import (
	"github.com/kubeedge/mapper-generator/pkg/common"
	"github.com/kubeedge/mapper-generator/pkg/global"
)

type DataBaseConfig struct {
}

func NewDataBaseClient() (global.DataBaseClient, error) {
	return &DataBaseConfig{}, nil
}

func (d *DataBaseConfig) InitDbClient() error {
	//TODO implement me
	panic("implement me")
}

func (d *DataBaseConfig) CloseSession() {
	//TODO implement me
	panic("implement me")
}

func (d *DataBaseConfig) AddData(data *common.DataModel) {
	//TODO implement me
	panic("implement me")
}

func (d *DataBaseConfig) GetDataByDeviceName(deviceName string) ([]*common.DataModel, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DataBaseConfig) GetPropertyDataByDeviceName(deviceName string, propertyData string) ([]*common.DataModel, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DataBaseConfig) GetDataByTimeRange(start int64, end int64) ([]*common.DataModel, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DataBaseConfig) DeleteDataByTimeRange(start int64, end int64) ([]*common.DataModel, error) {
	//TODO implement me
	panic("implement me")
}
