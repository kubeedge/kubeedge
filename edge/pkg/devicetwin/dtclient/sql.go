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

// InitDBTable create table
func InitDBTable(m core.Module) {
	klog.Info("Begin to register twin db model")

	if !m.Enable() {
		klog.Infof("Module %s is disabled, DB meta for it will not be registered", m.Name())
		return
	}
	orm.RegisterModel(new(Device))
	orm.RegisterModel(new(DeviceAttr))
	orm.RegisterModel(new(DeviceTwin))
}
