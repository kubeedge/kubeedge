package dtclient

import (
	"encoding/json"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

//DeviceTwin the struct of device twin
type DeviceTwin struct {
	ID              int64  `orm:"column(id);size(64);auto;pk"`
	DeviceName      string `orm:"column(device_name); null; type(text); unique"`
	DeviceNamespace string `orm:"column(device_namespace);null;type(text); unique"`
	PropertyName    string `orm:"column(property_name);null;type(text); unique"`
	Expected        string `orm:"column(expected);null;type(text)"`
	Actual          string `orm:"column(actual);null;type(text)"`
	ExpectedMeta    string `orm:"column(expected_meta);null;type(text)"`
	ActualMeta      string `orm:"column(actual_meta);null;type(text)"`
}

// TableUnique: name + namespace + propertyName as unique key
func (u *DeviceTwin) TableUnique() [][]string {
	return [][]string{
		{"device_name", "device_namespace", "property_name"},
	}
}

// DeviceTwinPrimaryKey: device_namespace + device_name + property_name as device_twin unique key
type DeviceTwinPrimaryKey struct {
	DeviceName      string
	DeviceNamespace string
	PropertyName    string
}

//DeviceTwinUpdate the struct for updating device twin
type DeviceTwinUpdate struct {
	DeviceName      string
	DeviceNamespace string
	PropertyName    string
	Cols            map[string]interface{}
}

// GetK8sDeviceFromDeviceTwin generate K8s Device from DeviceTwin
func GetK8sDeviceFromDeviceTwin(device []DeviceTwin) *v1alpha2.Device {
	if len(device) == 0 {
		return nil
	}
	result := &v1alpha2.Device{}
	result.Namespace = device[0].DeviceNamespace
	result.Name = device[0].DeviceName
	for _, v := range device {
		desiredMeta, _ := ConvertMetaToMap(v.ExpectedMeta)
		reportedMeta, _ := ConvertMetaToMap(v.ExpectedMeta)
		twin := v1alpha2.Twin{
			PropertyName: v.PropertyName,
			Desired: v1alpha2.TwinProperty{
				Value:    v.Expected,
				Metadata: desiredMeta,
			},
			Reported: v1alpha2.TwinProperty{
				Value:    v.Actual,
				Metadata: reportedMeta,
			},
		}
		result.Status.Twins = append(result.Status.Twins, twin)
	}
	return result
}

// GetDeviceTwinFromK8sDevice generate DeviceTwin from v1alpha2.Device
func GetDeviceTwinFromK8sDevice(device *v1alpha2.Device) []DeviceTwin {
	result := make([]DeviceTwin, len(device.Status.Twins))
	for index, twin := range device.Status.Twins {
		expectedMeta, _ := ConvertMetaMapToString(twin.Desired.Metadata)
		actualMeta, _ := ConvertMetaMapToString(twin.Reported.Metadata)
		result[index] = DeviceTwin{
			DeviceName:      device.Name,
			DeviceNamespace: device.Namespace,
			PropertyName:    twin.PropertyName,
			Expected:        twin.Desired.Value,
			ExpectedMeta:    expectedMeta,
			Actual:          twin.Reported.Value,
			ActualMeta:      actualMeta,
		}
	}

	return result
}

// ConvertMetaToMap convert string to map
func ConvertMetaToMap(metadata string) (map[string]string, error) {
	m := make(map[string]string)
	err := json.Unmarshal([]byte(metadata), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// ConvertMetaMapToString convert map to string
func ConvertMetaMapToString(metadata map[string]string) (string, error) {
	metaByte, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	return string(metaByte), nil
}

//SaveDeviceTwin save device twin
func SaveDeviceTwin(doc *DeviceTwin) error {
	num, err := dbm.DBAccess.Insert(doc)
	klog.V(4).Infof("Insert affected Num: %d, %s", num, err)
	return err
}

//DeleteDeviceTwinByDeviceID use name+namespace delete device all twins
func DeleteDeviceTwinByDeviceID(deviceKey *DevicePrimaryKey) error {
	num, err := dbm.DBAccess.QueryTable(DeviceTwinTableName).Filter("device_name", deviceKey.Name).Filter("device_namespace", deviceKey.Namespace).Delete()
	if err != nil {
		klog.Errorf("Something wrong when deleting data: %v", err)
		return err
	}
	klog.V(4).Infof("Delete affected Num: %d", num)
	return nil
}

//DeleteDeviceTwin delete device twin
func DeleteDeviceTwin(key *DeviceTwinPrimaryKey) error {
	num, err := dbm.DBAccess.QueryTable(DeviceTwinTableName).Filter("device_name", key.DeviceName).Filter("device_namespace", key.DeviceNamespace).Filter("property_name", key.PropertyName).Delete()
	if err != nil {
		klog.Errorf("Something wrong when deleting data: %v", err)
		return err
	}
	klog.V(4).Infof("Delete affected Num: %d", num)
	return nil
}

// UpdateDeviceTwinField update special field
func UpdateDeviceTwinField(key *DeviceTwinPrimaryKey, col string, value interface{}) error {
	num, err := dbm.DBAccess.QueryTable(DeviceTwinTableName).Filter("device_name", key.DeviceName).Filter("device_namespace", key.DeviceNamespace).Filter("property_name", key.PropertyName).Update(map[string]interface{}{col: value})
	klog.V(4).Infof("Update affected Num: %d, %s", num, err)
	return err
}

// UpdateDeviceTwinFields update special fields
func UpdateDeviceTwinFields(key *DeviceTwinPrimaryKey, cols map[string]interface{}) error {
	num, err := dbm.DBAccess.QueryTable(DeviceTwinTableName).Filter("device_name", key.DeviceName).Filter("device_namespace", key.DeviceNamespace).Filter("property_name", key.PropertyName).Update(cols)
	klog.Errorf("Update affected Num: %d, %v", num, err)
	klog.V(4).Infof("Update affected Num: %d, %v", num, err)
	return err
}

// QueryDeviceTwin query Device
func QueryDeviceTwin(key *DeviceTwinPrimaryKey) (*[]DeviceTwin, error) {
	twin := new([]DeviceTwin)
	_, err := dbm.DBAccess.QueryTable(DeviceTwinTableName).Filter("device_name", key.DeviceName).Filter("device_namespace", key.DeviceNamespace).All(twin)
	if err != nil {
		return nil, err
	}
	return twin, nil
}

//UpdateDeviceTwinMulti update device twin multi
func UpdateDeviceTwinMulti(updates []DeviceTwinUpdate) error {
	var err error
	for _, update := range updates {
		key := DeviceTwinPrimaryKey{
			DeviceNamespace: update.DeviceNamespace,
			DeviceName:      update.DeviceName,
			PropertyName:    update.PropertyName,
		}
		err = UpdateDeviceTwinFields(&key, update.Cols)
		if err != nil {
			return err
		}
	}
	return nil
}

//DeviceTwinTrans transaction of device twin
func DeviceTwinTrans(adds []DeviceTwin, deletes []DeviceTwinPrimaryKey, updates []DeviceTwinUpdate) error {
	var err error
	obm := dbm.DBAccess
	err = obm.Begin()
	for _, add := range adds {
		err = SaveDeviceTwin(&add)
		if err != nil {
			obm.Rollback()
			return err
		}
	}

	for _, delete := range deletes {
		deviceTwinPrimaryKey := &DeviceTwinPrimaryKey{
			PropertyName:    delete.PropertyName,
			DeviceName:      delete.DeviceName,
			DeviceNamespace: delete.DeviceNamespace,
		}
		err = DeleteDeviceTwin(deviceTwinPrimaryKey)
		if err != nil {
			obm.Rollback()
			return err
		}
	}

	for _, update := range updates {
		deviceTwinPrimaryKey := &DeviceTwinPrimaryKey{
			PropertyName:    update.PropertyName,
			DeviceName:      update.DeviceName,
			DeviceNamespace: update.DeviceNamespace,
		}
		err = UpdateDeviceTwinFields(deviceTwinPrimaryKey, update.Cols)
		if err != nil {
			obm.Rollback()
			return err
		}
	}
	obm.Commit()
	return nil
}
