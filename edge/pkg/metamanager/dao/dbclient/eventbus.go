package dbclient

import (
	"errors"

	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

type EventBusService struct {
	db *gorm.DB
}

func NewEventBusService() *EventBusService {
	return &EventBusService{db: dao.GetDB()}
}

// InsertTopics insert or replace topic into sub_topics
func (s *EventBusService) InsertTopics(topic string) error {
	// GORM upsert using SQLite syntax: INSERT OR REPLACE
	err := s.db.Exec("INSERT OR REPLACE INTO sub_topics (topic) VALUES (?)", topic).Error
	if err != nil {
		klog.Errorf("Failed to insert or replace topic: %v", err)
	}
	return err
}

// DeleteTopicsByKey deletes a topic from sub_topics
func (s *EventBusService) DeleteTopicsByKey(key string) error {
	result := s.db.Delete(&models.SubTopics{}, "topic = ?", key)
	if result.Error != nil {
		klog.Errorf("Failed to delete topic: %v", result.Error)
		return result.Error
	}
	klog.V(4).Infof("Delete affected Num: %d", result.RowsAffected)
	return nil
}

// QueryAllTopics retrieves all topics from sub_topics
func (s *EventBusService) QueryAllTopics() (*[]string, error) {
	var entries []models.SubTopics
	if err := s.db.Find(&entries).Error; err != nil {
		klog.Errorf("Failed to query topics: %v", err)
		return nil, err
	}
	if len(entries) == 0 {
		return nil, errors.New("no topics found")
	}
	var result []string
	for _, entry := range entries {
		result = append(result, entry.Topic)
	}
	return &result, nil
}
