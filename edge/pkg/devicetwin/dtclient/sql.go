package dtclient

import (
	"github.com/astaxie/beego/orm"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
)

const (
	//DeviceTableName device table
	DeviceTableName = "device"
	//DeviceAttrTableName device table
	DeviceAttrTableName = "device_attr"
	//DeviceTwinTableName device table
	DeviceTwinTableName = "device_twin"
	//DeviceMetaTableName device table
	DeviceMetaTableName = "device_meta"
)

// InitDBTable create table
func InitDBTable(module core.Module) {
	klog.Infof("Begin to register %v db model", module.Name())

	if !module.Enable() {
		klog.Infof("Module %s is disabled, DB meta for it will not be registered", module.Name())
		return
	}
	orm.RegisterModel(new(Device))
	orm.RegisterModel(new(DeviceAttr))
	orm.RegisterModel(new(DeviceTwin))
	orm.RegisterModel(new(DeviceMeta))
}
