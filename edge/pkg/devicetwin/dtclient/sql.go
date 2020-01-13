package dtclient

import (
	"github.com/astaxie/beego/orm"
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/common/constants"
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
func InitDBTable() {
	klog.Info("Begin to register twin db model")

	if !core.IsModuleEnabled(constants.DeviceTwinModuleName) {
		klog.Infof("Module %s is disabled, DB meta for it will not be registered", constants.DeviceTwinModuleName)
		return
	}
	orm.RegisterModel(new(Device))
	orm.RegisterModel(new(DeviceAttr))
	orm.RegisterModel(new(DeviceTwin))
}
