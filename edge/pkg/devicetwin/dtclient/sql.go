package dtclient

import (
	"github.com/astaxie/beego/orm"
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
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
	klog.Info("Begin to register twin db model")

	if !core.IsModuleEnabled("twin") {
		klog.Info("DB meta for twin module has not been registered,so can not init db table")
		return
	}
	orm.RegisterModel(new(Device))
	orm.RegisterModel(new(DeviceAttr))
	orm.RegisterModel(new(DeviceTwin))
}
