package dtclient

import (
	"github.com/beego/beego/v2/client/orm"
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
func SaveDevice(to orm.TxOrmer, doc *Device) error {
	num, err := to.Insert(doc)
	klog.V(4).Infof("Insert affected Num: %d, %v", num, err)
	return err
}

// DeleteDeviceByID delete device by id
func DeleteDeviceByID(to orm.TxOrmer, id string) error {
	num, err := to.QueryTable(DeviceTableName).Filter("id", id).Delete()
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

// AddDeviceTrans the transaction of add device
func AddDeviceTrans(adds []Device, addAttrs []DeviceAttr, addTwins []DeviceTwin) error {
	obm := dbm.DefaultOrmFunc()
	to, err := obm.Begin() // In v2 version of beego/beego, it is recommended to use specific transactions to perform database inserts instead of global inserts.
	if err != nil {
		klog.Errorf("failed to begin transaction: %v", err)
		return err
	}

	defer func() {
		if err != nil {
			dbm.RollbackTransaction(to)
		} else {
			err = to.Commit()
			if err != nil {
				klog.Errorf("failed to commit transaction: %v", err)
			}
		}
	}()

	for _, add := range adds {
		err = SaveDevice(to, &add)

		if err != nil {
			klog.Errorf("save device failed: %v", err)
			return err
		}
	}

	for _, attr := range addAttrs {
		err = SaveDeviceAttr(to, &attr)
		if err != nil {
			return err
		}
	}

	for _, twin := range addTwins {
		err = SaveDeviceTwin(to, &twin)
		if err != nil {
			return err
		}
	}

	return err
}

// DeleteDeviceTrans the transaction of delete device
func DeleteDeviceTrans(deletes []string) error {
	obm := dbm.DefaultOrmFunc()
	to, err := obm.Begin() // In v2 version of beego/beego, it is recommended to use specific transactions to perform database delete instead of global delete.
	if err != nil {
		klog.Errorf("failed to begin transaction: %v", err)
		return err
	}

	defer func() {
		if err != nil {
			dbm.RollbackTransaction(to)
		} else {
			err = to.Commit()
			if err != nil {
				klog.Errorf("failed to commit transaction: %v", err)
			}
		}
	}()

	for _, delete := range deletes {
		err = DeleteDeviceByID(to, delete)
		if err != nil {
			return err
		}
		err = DeleteDeviceAttrByDeviceID(to, delete)
		if err != nil {
			return err
		}
		err = DeleteDeviceTwinByDeviceID(to, delete)
		if err != nil {
			return err
		}
	}

	return err
}
