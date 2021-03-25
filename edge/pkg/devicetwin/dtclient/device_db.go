package dtclient

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

//Device the struct of device
type Device struct {
	ID        int64  `orm:"column(id); size(64); auto; pk"`
	Name      string `orm:"column(name); null; type(text); unique"`
	Namespace string `orm:"column(namespace); null; type(text); unique"`
}

// TableUnique name + namespace as unique key
func (u *Device) TableUnique() [][]string {
	return [][]string{
		{"name", "namespace"},
	}
}

// DevicePrimaryKey table device composite key
type DevicePrimaryKey struct {
	Name      string
	Namespace string
}

//DeviceUpdate the struct for updating device
type DeviceUpdate struct {
	Name      string
	Namespace string
	Cols      map[string]interface{}
}

func ConvertCloudDeviceToTableDevice(device v1alpha2.Device) Device {
	result := Device{}

	result.Name = device.Name
	result.Namespace = device.Namespace
	return result
}

//SaveDevice save device
func SaveDevice(doc *Device) error {
	num, err := dbm.DBAccess.Insert(doc)
	klog.V(4).Infof("Insert affected Num: %d, %v", num, err)
	return err
}

//DeleteDeviceByKey delete device by primary key
func DeleteDeviceByKey(key DevicePrimaryKey) error {
	num, err := dbm.DBAccess.QueryTable(DeviceTableName).Filter("name", key.Name).Filter("namespace", key.Namespace).Delete()
	if err != nil {
		klog.Errorf("Something wrong when deleting data: %v", err)
		return err
	}
	klog.V(4).Infof("Delete affected Num: %d", num)
	return nil
}

// UpdateDeviceField update special field
func UpdateDeviceField(deviceKey DevicePrimaryKey, col string, value interface{}) error {
	num, err := dbm.DBAccess.QueryTable(DeviceTableName).Filter("name", deviceKey.Name).Filter("namespace", deviceKey.Namespace).Update(map[string]interface{}{col: value})
	klog.V(4).Infof("Update affected Num: %d, %s", num, err)
	return err
}

// UpdateDeviceFields update special fields
func UpdateDeviceFields(deviceKey DevicePrimaryKey, cols map[string]interface{}) error {
	num, err := dbm.DBAccess.QueryTable(DeviceTableName).Filter("name", deviceKey.Name).Filter("namespace", deviceKey.Namespace).Update(cols)
	klog.V(4).Infof("Update affected Num: %d, %s", num, err)
	return err
}

// QueryDeviceByKey query Device
func QueryDeviceByKey(primaryKey DevicePrimaryKey) (*[]Device, error) {
	devices := new([]Device)
	_, err := dbm.DBAccess.QueryTable(DeviceTableName).Filter("name", primaryKey.Name).Filter("namespace", primaryKey.Namespace).All(devices)
	if err != nil {
		return nil, err
	}
	return devices, nil
}

// QueryDeviceAll query twin
func QueryDeviceAll() (*[]Device, error) {
	devices := new([]Device)
	_, err := dbm.DBAccess.QueryTable(DeviceTableName).All(devices)
	if err != nil {
		return nil, err
	}
	return devices, nil
}

//UpdateDeviceMulti update device  multi
func UpdateDeviceMulti(updates []DeviceUpdate) error {
	var err error
	for _, update := range updates {
		primaryKey := DevicePrimaryKey{
			Name:      update.Name,
			Namespace: update.Namespace,
		}
		err = UpdateDeviceFields(primaryKey, update.Cols)
		if err != nil {
			return err
		}
	}
	return nil
}

//AddDeviceTrans the transaction of add device
func AddDeviceTrans(adds []Device, addTwins []DeviceTwin) error {
	var err error
	obm := dbm.DBAccess
	obm.Begin()
	for _, add := range adds {
		err = SaveDevice(&add)

		if err != nil {
			klog.Errorf("save device %v failed: %v", add, err)
			obm.Rollback()
			return err
		}
	}

	for _, twin := range addTwins {
		err = SaveDeviceTwin(&twin)
		if err != nil {
			obm.Rollback()
			return err
		}
	}
	obm.Commit()
	return nil
}

//DeleteDeviceTrans the transaction of delete device
func DeleteDeviceTrans(deletes []DevicePrimaryKey) error {
	var err error
	obm := dbm.DBAccess
	obm.Begin()
	for _, delete := range deletes {
		err = DeleteDeviceByKey(delete)
		if err != nil {
			obm.Rollback()
			return err
		}

		klog.Infof("delete device twin from table, %v", delete)
		err = DeleteDeviceTwinByDeviceID(&delete)
		if err != nil {
			klog.Errorf("delete device twin from table, err is %v", err)
			obm.Rollback()
			return err
		}
	}
	obm.Commit()
	return nil
}
