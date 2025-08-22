package dbclient

import (
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"

	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

type MetaV2Service struct {
	db *gorm.DB
}

func NewMetaV2Service() *MetaV2Service {
	return &MetaV2Service{db: dao.GetDB()}
}

func (s *MetaV2Service) RawMetaByGVRNN(gvr schema.GroupVersionResource, namespace string, name string) (*[]models.MetaV2, error) {
	var objs []models.MetaV2
	tx := s.db.Model(&models.MetaV2{})

	if !gvr.Empty() {
		tx = tx.Where(models.GVR+" = ?", gvr.String())
	}

	if namespace != models.NullNamespace && namespace != "" {
		tx = tx.Where(models.NS+" = ?", namespace)
	}

	if name != models.NullName && name != "" {
		tx = tx.Where(models.NAME+" = ?", name)
	}

	if err := tx.Find(&objs).Error; err != nil {
		return nil, err
	}
	return &objs, nil
}

func (s *MetaV2Service) GetLatestMetaV2() (models.MetaV2, error) {
	var meta models.MetaV2
	err := s.db.Model(&models.MetaV2{}).Order(clause.OrderByColumn{Column: clause.Column{Name: models.RV}, Desc: true}).Limit(1).Find(&meta).Error
	return meta, err
}

func (s *MetaV2Service) InsertOrReplaceMetaV2(m *models.MetaV2) error {
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		UpdateAll: true,
	}).Create(m).Error
}

func (s *MetaV2Service) RetryInsertOrReplaceMetaV2(m *models.MetaV2, maxRetries int) error {
	err := s.InsertOrReplaceMetaV2(m)
	if err == nil {
		return nil
	}
	for i := 1; i < maxRetries; i++ {
		klog.Errorf("retry %d: failed to insert or replace meta_v2: %v", i, err)
		err = s.InsertOrReplaceMetaV2(m)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("failed to insert or replace meta_v2 after %d retries: %v", maxRetries, err)
}

func (s *MetaV2Service) GetByKey(key string) (*models.MetaV2, error) {
	var result models.MetaV2
	err := s.db.Where(models.KEY+" = ?", key).First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *MetaV2Service) DeleteByKey(key string) error {
	return s.db.Where(models.KEY+" = ?", key).Delete(&models.MetaV2{}).Error
}

// upgrade_db
func (s *MetaV2Service) SaveNodeUpgradeJobRequestToMetaV2(nodeUpgradeJobReq commontypes.NodeUpgradeJobRequest) error {
	db := s.db

	nodeUpgradeJobReqJSON, err := json.Marshal(nodeUpgradeJobReq)
	if err != nil {
		return errors.New("failed to marshal NodeUpgradeJobRequest")
	}

	meta := models.MetaV2{
		Key:   nodeUpgradeJobReq.UpgradeID,
		Name:  models.NodeUpgradeJobRequestName,
		Value: string(nodeUpgradeJobReqJSON),
	}

	var count int64
	if err := db.Model(&models.MetaV2{}).
		Where(models.NAME+" = ?", models.NodeUpgradeJobRequestName).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		klog.Info("NodeUpgradeJobRequest exists, updating...")
		return db.Model(&models.MetaV2{}).
			Where(models.NAME+" = ?", models.NodeUpgradeJobRequestName).
			Updates(meta).Error
	}

	klog.Info("NodeUpgradeJobRequest not found, inserting...")
	return db.Create(&meta).Error
}

func (s *MetaV2Service) DeleteNodeUpgradeJobRequestFromMetaV2() error {
	return s.db.Where(models.NAME+" = ?", models.NodeUpgradeJobRequestName).Delete(&models.MetaV2{}).Error
}

func (s *MetaV2Service) QueryNodeUpgradeJobRequestFromMetaV2() (commontypes.NodeUpgradeJobRequest, error) {
	var nodeUpgradeReq commontypes.NodeUpgradeJobRequest
	var meta models.MetaV2

	err := s.db.Where(models.NAME+" = ?", models.NodeUpgradeJobRequestName).First(&meta).Error
	if err != nil {
		return nodeUpgradeReq, err
	}

	err = json.Unmarshal([]byte(meta.Value), &nodeUpgradeReq)
	if err != nil {
		return nodeUpgradeReq, fmt.Errorf("failed to unmarshal NodeUpgradeJobRequest: %v", err)
	}
	return nodeUpgradeReq, nil
}

func (s *MetaV2Service) checkIfNodeTaskRequestExists() bool {
	var count int64
	s.db.Model(&models.MetaV2{}).Where(models.NAME+" = ?", models.NodeTaskRequestName).Count(&count)
	return count > 0
}

func (s *MetaV2Service) SaveNodeTaskRequestToMetaV2(nodeTaskReq commontypes.NodeTaskRequest) error {
	db := s.db
	nodeTaskReqJSON, err := json.Marshal(nodeTaskReq)
	if err != nil {
		return errors.New("failed to marshal NodeTaskRequest")
	}

	meta := models.MetaV2{
		Key:   nodeTaskReq.TaskID,
		Name:  models.NodeTaskRequestName,
		Value: string(nodeTaskReqJSON),
	}

	if s.checkIfNodeTaskRequestExists() {
		klog.Info("NodeTaskRequest exists, updating...")
		return db.Model(&models.MetaV2{}).Where(models.NAME+" = ?", models.NodeTaskRequestName).Updates(meta).Error
	} else {
		klog.Info("NodeTaskRequest not found, inserting...")
		return db.Create(&meta).Error
	}
}

func (s *MetaV2Service) DeleteNodeTaskRequestFromMetaV2() error {
	return s.db.Where(models.NAME+" = ?", models.NodeTaskRequestName).Delete(&models.MetaV2{}).Error
}

func (s *MetaV2Service) QueryNodeTaskRequestFromMetaV2() (commontypes.NodeTaskRequest, error) {
	var nodeTaskReq commontypes.NodeTaskRequest
	var meta models.MetaV2

	err := s.db.Where(models.NAME+" = ?", models.NodeTaskRequestName).First(&meta).Error
	if err != nil {
		return nodeTaskReq, err
	}

	err = json.Unmarshal([]byte(meta.Value), &nodeTaskReq)
	if err != nil {
		return nodeTaskReq, fmt.Errorf("failed to unmarshal NodeTaskRequest: %v", err)
	}
	return nodeTaskReq, nil
}
