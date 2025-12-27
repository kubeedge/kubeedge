package dbclient

import (
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

type DeviceService struct {
	db *gorm.DB
}

func NewDeviceService() *DeviceService {
	return &DeviceService{db: dao.GetDB()}
}

// SaveDevice saves a device in a transaction
func (s *DeviceService) SaveDevice(doc *models.Device) error {
	tx := s.db
	if err := tx.Create(doc).Error; err != nil {
		tx.Rollback()
		klog.Errorf("Failed to insert Device data: %v", err)
		return err
	}
	return tx.Commit().Error
}

// DeleteDeviceByID deletes a device by ID in a transaction
func (s *DeviceService) DeleteDeviceByID(id string) error {
	tx := s.db.Begin()
	if err := tx.Delete(&models.Device{}, "id = ?", id).Error; err != nil {
		tx.Rollback()
		klog.Errorf("Failed to delete Device data: %v", err)
		return err
	}
	return tx.Commit().Error
}

// UpdateDeviceField updates a single column
func (s *DeviceService) UpdateDeviceField(deviceID string, col string, value interface{}) error {
	err := s.db.Model(&models.Device{}).Where("id = ?", deviceID).Update(col, value).Error
	klog.V(4).Infof("Update affected for field %s: %v", col, err)
	return err
}

// UpdateDeviceFields updates multiple columns
func (s *DeviceService) UpdateDeviceFields(deviceID string, cols map[string]interface{}) error {
	err := s.db.Model(&models.Device{}).Where("id = ?", deviceID).Updates(cols).Error
	klog.V(4).Infof("Update affected multiple fields: %v", err)
	return err
}

// QueryDevice queries devices with a condition
func (s *DeviceService) QueryDevice(key string, condition string) ([]models.Device, error) {
	var devices []models.Device
	err := s.db.Where(key+" = ?", condition).Find(&devices).Error
	if err != nil {
		klog.Errorf("Failed to query device: %v", err)
		return nil, err
	}
	return devices, nil
}

// QueryDeviceAll returns all devices
func (s *DeviceService) QueryDeviceAll() ([]models.Device, error) {
	var devices []models.Device
	err := s.db.Find(&devices).Error
	if err != nil {
		klog.Errorf("Failed to query all devices: %v", err)
		return nil, err
	}
	return devices, nil
}

// UpdateDeviceMulti updates multiple devices
func (s *DeviceService) UpdateDeviceMulti(updates []models.DeviceUpdate) error {
	for _, update := range updates {
		if err := s.UpdateDeviceFields(update.DeviceID, update.Cols); err != nil {
			return err
		}
	}
	return nil
}

// AddDeviceTrans handles transactional creation of devices, attributes, and twins
func (s *DeviceService) AddDeviceTrans(adds []models.Device, addAttrs []models.DeviceAttr, addTwins []models.DeviceTwin) error {
	tx := s.db

	for _, device := range adds {
		if err := tx.Create(&device).Error; err != nil {
			tx.Rollback()
			klog.Errorf("Failed to save device: %v", err)
			return err
		}
	}

	for _, attr := range addAttrs {
		if err := tx.Create(&attr).Error; err != nil {
			tx.Rollback()
			klog.Errorf("Failed to save device attr: %v", err)
			return err
		}
	}

	for _, twin := range addTwins {
		if err := tx.Create(&twin).Error; err != nil {
			tx.Rollback()
			klog.Errorf("Failed to save device twin: %v", err)
			return err
		}
	}

	return tx.Commit().Error
}

// DeleteDeviceTrans handles transactional deletion of devices, attributes, and twins
func (s *DeviceService) DeleteDeviceTrans(deletes []string) error {
	tx := s.db

	for _, id := range deletes {
		if err := tx.Delete(&models.Device{}, "id = ?", id).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := s.DeleteDeviceAttrByDeviceID(id); err != nil {
			tx.Rollback()
			return err
		}
		if err := s.DeleteDeviceTwinByDeviceID(id); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (s *DeviceService) SaveDeviceAttr(doc *models.DeviceAttr) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(doc).Error; err != nil {
			klog.Errorf("Failed to insert DeviceAttr: %v", err)
			return err
		}
		klog.V(4).Info("Inserted DeviceAttr successfully")
		return nil
	})
}

func (s *DeviceService) DeleteDeviceAttrByDeviceID(deviceID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("deviceid = ?", deviceID).Delete(&models.DeviceAttr{}).Error; err != nil {
			klog.Errorf("Failed to delete DeviceAttr by deviceID: %v", err)
			return err
		}
		return nil
	})
}

func (s *DeviceService) DeleteDeviceAttr(deviceID string, name string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("deviceid = ? AND name = ?", deviceID, name).Delete(&models.DeviceAttr{}).Error; err != nil {
			klog.Errorf("Failed to delete DeviceAttr by deviceID and name: %v", err)
			return err
		}
		return nil
	})
}

func (s *DeviceService) UpdateDeviceAttrField(deviceID, name, col string, value interface{}) error {
	result := s.db.Model(&models.DeviceAttr{}).Where("deviceid = ? AND name = ?", deviceID, name).Update(col, value)
	klog.V(4).Infof("Update affected rows: %d, error: %v", result.RowsAffected, result.Error)
	return result.Error
}

func (s *DeviceService) UpdateDeviceAttrFields(deviceID, name string, cols map[string]interface{}) error {
	result := s.db.Model(&models.DeviceAttr{}).Where("deviceid = ? AND name = ?", deviceID, name).Updates(cols)
	klog.V(4).Infof("Update affected rows: %d, error: %v", result.RowsAffected, result.Error)
	return result.Error
}

func (s *DeviceService) QueryDeviceAttr(key, condition string) (*[]models.DeviceAttr, error) {
	var attrs []models.DeviceAttr
	if err := s.db.Where(key+" = ?", condition).Find(&attrs).Error; err != nil {
		return nil, err
	}
	return &attrs, nil
}

func (s *DeviceService) UpdateDeviceAttrMulti(updates []models.DeviceAttrUpdate) error {
	for _, update := range updates {
		if err := s.UpdateDeviceAttrFields(update.DeviceID, update.Name, update.Cols); err != nil {
			return err
		}
	}
	return nil
}

func (s *DeviceService) DeviceAttrTrans(adds []models.DeviceAttr, deletes []models.DeviceDelete, updates []models.DeviceAttrUpdate) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, add := range adds {
			if err := tx.Create(&add).Error; err != nil {
				return err
			}
		}
		for _, del := range deletes {
			if err := tx.Where("deviceid = ? AND name = ?", del.DeviceID, del.Name).Delete(&models.DeviceAttr{}).Error; err != nil {
				return err
			}
		}
		for _, upd := range updates {
			if err := tx.Model(&models.DeviceAttr{}).
				Where("deviceid = ? AND name = ?", upd.DeviceID, upd.Name).
				Updates(upd.Cols).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *DeviceService) SaveDeviceTwin(doc *models.DeviceTwin) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(doc).Error; err != nil {
			klog.Errorf("Failed to insert DeviceTwin: %v", err)
			return err
		}
		klog.V(4).Info("Inserted DeviceTwin successfully")
		return nil
	})
}

func (s *DeviceService) DeleteDeviceTwinByDeviceID(deviceID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("deviceid = ?", deviceID).Delete(&models.DeviceTwin{}).Error; err != nil {
			klog.Errorf("Failed to delete DeviceTwin by deviceID: %v", err)
			return err
		}
		return nil
	})
}

func (s *DeviceService) DeleteDeviceTwin(deviceID, name string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("deviceid = ? AND name = ?", deviceID, name).Delete(&models.DeviceTwin{}).Error; err != nil {
			klog.Errorf("Failed to delete DeviceTwin: %v", err)
			return err
		}
		return nil
	})
}

func (s *DeviceService) UpdateDeviceTwinField(deviceID, name, col string, value interface{}) error {
	result := s.db.Model(&models.DeviceTwin{}).Where("deviceid = ? AND name = ?", deviceID, name).Update(col, value)
	klog.V(4).Infof("Update affected rows: %d, error: %v", result.RowsAffected, result.Error)
	return result.Error
}

func (s *DeviceService) UpdateDeviceTwinFields(deviceID, name string, cols map[string]interface{}) error {
	result := s.db.Model(&models.DeviceTwin{}).Where("deviceid = ? AND name = ?", deviceID, name).Updates(cols)
	klog.V(4).Infof("Update affected rows: %d, error: %v", result.RowsAffected, result.Error)
	return result.Error
}

func (s *DeviceService) QueryDeviceTwin(key, condition string) (*[]models.DeviceTwin, error) {
	var twins []models.DeviceTwin
	if err := s.db.Where(key+" = ?", condition).Find(&twins).Error; err != nil {
		return nil, err
	}
	return &twins, nil
}

func (s *DeviceService) UpdateDeviceTwinMulti(updates []models.DeviceTwinUpdate) error {
	for _, update := range updates {
		if err := s.UpdateDeviceTwinFields(update.DeviceID, update.Name, update.Cols); err != nil {
			return err
		}
	}
	return nil
}

func (s *DeviceService) DeviceTwinTrans(adds []models.DeviceTwin, deletes []models.DeviceDelete, updates []models.DeviceTwinUpdate) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, add := range adds {
			if err := tx.Create(&add).Error; err != nil {
				return err
			}
		}
		for _, del := range deletes {
			if err := tx.Where("deviceid = ? AND name = ?", del.DeviceID, del.Name).Delete(&models.DeviceTwin{}).Error; err != nil {
				return err
			}
		}
		for _, upd := range updates {
			if err := tx.Model(&models.DeviceTwin{}).
				Where("deviceid = ? AND name = ?", upd.DeviceID, upd.Name).
				Updates(upd.Cols).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
