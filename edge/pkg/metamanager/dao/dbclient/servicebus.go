package dbclient

import (
	"errors"

	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

type ServiceBusService struct {
	db *gorm.DB
}

func NewServiceBusService() *ServiceBusService {
	return &ServiceBusService{db: dao.GetDB()}
}

// InsertUrls insert or replace url into target_urls
func (s *ServiceBusService) InsertUrls(url string) error {
	// Raw SQL query to perform INSERT OR REPLACE for SQLite
	err := s.db.Exec("INSERT OR REPLACE INTO target_urls (url) VALUES (?)", url).Error
	if err != nil {
		klog.Errorf("Failed to insert or replace URL: %v", err)
	}
	return err
}

// DeleteUrlsByKey delete target_urls by URL
func (s *ServiceBusService) DeleteUrlsByKey(key string) error {
	result := s.db.Where("url = ?", key).Delete(&models.TargetUrls{})
	if result.Error != nil {
		klog.Errorf("Failed to delete URL %s: %v", key, result.Error)
		return result.Error
	}

	klog.V(4).Infof("Delete affected Num: %d", result.RowsAffected)
	return nil
}

// IsTableEmpty returns true if no records in target_urls
func (s *ServiceBusService) IsTableEmpty() bool {
	var count int64
	err := s.db.Model(&models.TargetUrls{}).Count(&count).Error
	if err != nil {
		klog.Errorf("Failed to count target_urls: %v", err)
		return true
	}
	return count == 0
}

// GetUrlsByKey gets one record from target_urls by URL
func (s *ServiceBusService) GetUrlsByKey(key string) (*models.TargetUrls, error) {
	var target models.TargetUrls
	err := s.db.Where("url = ?", key).First(&target).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			klog.V(4).Infof("URL not found: %s", key)
			return nil, nil
		}
		klog.Errorf("Failed to get URL %s: %v", key, err)
		return nil, err
	}

	return &target, nil
}
