package dtclient

import (
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

const (
	//DeviceTableName device table
	DeviceTableName = "device"
	//DeviceAttrTableName device table
	DeviceAttrTableName = "device_attr"
	//DeviceTwinTableName device table
	DeviceTwinTableName = "device_twin"
)

//InitDBTable create table
func InitDBTable() {
	klog.Info("Begin to register twin model")
	dbm.RegisterModel("twin", new(Device))
	dbm.RegisterModel("twin", new(DeviceAttr))
	dbm.RegisterModel("twin", new(DeviceTwin))
}
