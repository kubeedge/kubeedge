package dtclient

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

// Device the struct of device
type Device struct {
	ID          string `orm:"column(id); size(64); pk"`
	Name        string `orm:"column(name); null; type(text)"`
	Description string `orm:"column(description); null; type(text)"`
	State       string `orm:"column(state); null; type(text)"`
	LastOnline  string `orm:"column(last_online); null; type(text)"`
}

// SaveDevice save device
func SaveDevice(doc *Device) error {
	num, err := dbm.DBAccess.Insert(doc)
	klog.V(4).Infof("Insert affected Num: %d, %v", num, err)
	return err
}

// DeleteDeviceByID delete device by id
func DeleteDeviceByID(id string) error {
	num, err := dbm.DBAccess.QueryTable(DeviceTableName).Filter("id", id).Delete()
	if err != nil {
		klog.Errorf("Something wrong when deleting data: %v", err)
		return err
	}
	klog.V(4).Infof("Delete affected Num: %d", num)
	return nil
}

// UpdateDeviceField update special field
func UpdateDeviceField(deviceID string, col string, value interface{}) error {
	num, err := dbm.DBAccess.QueryTable(DeviceTableName).Filter("id", deviceID).Update(map[string]interface{}{col: value})
	klog.V(4).Infof("Update affected Num: %d, %s", num, err)
	return err
}

// UpdateDeviceFields update special fields
func UpdateDeviceFields(deviceID string, cols map[string]interface{}) error {
	num, err := dbm.DBAccess.QueryTable(DeviceTableName).Filter("id", deviceID).Update(cols)
	klog.V(4).Infof("Update affected Num: %d, %s", num, err)
	return err
}

// QueryDevice query Device
func QueryDevice(key string, condition string) (*[]Device, error) {
	devices := new([]Device)
	_, err := dbm.DBAccess.QueryTable(DeviceTableName).Filter(key, condition).All(devices)
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

// DeviceUpdate the struct for updating device
type DeviceUpdate struct {
	DeviceID string
	Cols     map[string]interface{}
}

// UpdateDeviceMulti update device  multi
func UpdateDeviceMulti(updates []DeviceUpdate) error {
	var err error
	for _, update := range updates {
		err = UpdateDeviceFields(update.DeviceID, update.Cols)
		if err != nil {
			return err
		}
	}
	return nil
}

//AddDeviceTrans add device with transaction
func AddDeviceTrans(adds []Device, addAttrs []DeviceAttr, addTwins []DeviceTwin) error {
	var err error
	obm := dbm.DBAccess
	err = obm.Begin()
	if err != nil {
		klog.Errorf("begin transaction failed: %v", err)
		return err
	}
	for _, add := range adds {
		err = SaveDevice(&add)

		if err != nil {
			klog.Errorf("save device failed: %v", err)
			// we do not handle the error of rollback because:
			// 1. no matter whether the rollback is successful or not,
			// the transaction will not be committed.
			// 2. err of SaveDevice is the main err that why AddDeviceTrans exit,
			// so we should the SaveDevice err.
			obm.Rollback()
			return err
		}
	}

	for _, attr := range addAttrs {
		err = SaveDeviceAttr(&attr)
		if err != nil {
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
	return obm.Commit()
}

// DeleteDeviceTrans  delete device with transaction
func DeleteDeviceTrans(deletes []string) error {
	var err error
	obm := dbm.DBAccess
	err = obm.Begin()
	if err != nil {
		klog.Errorf("begin transaction failed: %v", err)
		return err
	}
	for _, deleteID := range deletes {
		err = DeleteDeviceByID(deleteID)
		if err != nil {
			obm.Rollback()
			return err
		}
		err = DeleteDeviceAttrByDeviceID(deleteID)
		if err != nil {
			obm.Rollback()
			return err
		}
		err = DeleteDeviceTwinByDeviceID(deleteID)
		if err != nil {
			obm.Rollback()
			return err
		}
	}
	return obm.Commit()
}
