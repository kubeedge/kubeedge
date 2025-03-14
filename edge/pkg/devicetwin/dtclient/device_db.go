package dtclient

import (
	"context"

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
func SaveDevice(o orm.Ormer, doc *Device) error {
	err := o.DoTx(func(_ context.Context, txOrm orm.TxOrmer) error {
		// insert data
		// Using txOrm to execute SQL
		_, e := txOrm.Insert(doc)
		// if e != nil the transaction will be rollback
		// or it will be committed
		return e
	})
	if err != nil {
		klog.Errorf("Something wrong when insert Device data: %v", err)
		return err
	}
	klog.V(4).Info("insert Device data successfully")
	return nil
}

// DeleteDeviceByID delete device by id
func DeleteDeviceByID(o orm.Ormer, id string) error {
	err := o.DoTx(func(_ context.Context, txOrm orm.TxOrmer) error {
		// delete data
		// Using txOrm to execute SQL
		_, e := txOrm.QueryTable(DeviceTableName).Filter("id", id).Delete()
		// if e != nil the transaction will be rollback
		// or it will be committed
		return e
	})

	if err != nil {
		klog.Errorf("Something wrong when deleting Device data: %v", err)
		return err
	}
	klog.V(4).Info("Delete Device data successfully")
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
	var err error
	for _, add := range adds {
		err = SaveDevice(obm, &add)
		if err != nil {
			klog.Errorf("save device failed: %v", err)
			return err
		}
	}

	for _, attr := range addAttrs {
		err = SaveDeviceAttr(obm, &attr)
		if err != nil {
			klog.Errorf("save device attr failed: %v", err)
			return err
		}
	}

	for _, twin := range addTwins {
		err = SaveDeviceTwin(obm, &twin)
		if err != nil {
			klog.Errorf("save device twin failed: %v", err)
			return err
		}
	}

	return err
}

// DeleteDeviceTrans the transaction of delete device
func DeleteDeviceTrans(deletes []string) error {
	obm := dbm.DefaultOrmFunc()
	var err error

	for _, delete := range deletes {
		err = DeleteDeviceByID(obm, delete)
		if err != nil {
			return err
		}
		err = DeleteDeviceAttrByDeviceID(obm, delete)
		if err != nil {
			return err
		}
		err = DeleteDeviceTwinByDeviceID(obm, delete)
		if err != nil {
			return err
		}
	}

	return err
}
