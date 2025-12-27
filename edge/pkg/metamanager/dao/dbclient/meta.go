package dbclient

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

type MetaService struct {
	db *gorm.DB
}

func NewMetaService() *MetaService {
	return &MetaService{db: dao.GetDB()}
}

// SaveMeta saves meta to db
func (s *MetaService) SaveMeta(meta *models.Meta) error {
	err := s.db.Create(meta).Error
	if err == nil || IsNonUniqueNameError(err) {
		return nil
	}
	return err
}

// IsNonUniqueNameError tests if the error is due to a unique constraint violation.
func IsNonUniqueNameError(err error) bool {
	if err == nil {
		return false
	}
	str := err.Error()
	return strings.HasSuffix(str, "are not unique") ||
		strings.Contains(str, "UNIQUE constraint failed") ||
		strings.HasSuffix(str, "constraint failed")
}

// DeleteMetaByKey deletes meta by key
func (s *MetaService) DeleteMetaByKey(key string) error {
	return s.db.Where("key = ?", key).Delete(&models.Meta{}).Error
}

// DeleteMetaByKeyAndPodUID deletes meta by key and podUID from value field
func (s *MetaService) DeleteMetaByKeyAndPodUID(key, podUID string) (int64, error) {
	result := s.db.Where("key = ? AND value LIKE ?", key, "%"+podUID+"%").Delete(&models.Meta{})
	if result.Error != nil {
		klog.Errorf("delete pod by key %s and podUID %s failed, err: %v", key, podUID, result.Error)
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// UpdateMeta updates a meta entry (all fields)
func (s *MetaService) UpdateMeta(meta *models.Meta) error {
	return s.db.Save(meta).Error
}

// InsertOrUpdate inserts or replaces a meta entry
func (s *MetaService) InsertOrUpdate(meta *models.Meta) error {
	// SQLite syntax
	sql := "INSERT OR REPLACE INTO meta (key, type, value) VALUES (?, ?, ?)"
	return s.db.Exec(sql, meta.Key, meta.Type, meta.Value).Error
}

// UpdateMetaField updates one field
func (s *MetaService) UpdateMetaField(key string, col string, value interface{}) error {
	return s.db.Model(&models.Meta{}).Where("key = ?", key).Update(col, value).Error
}

// UpdateMetaFields updates multiple fields
func (s *MetaService) UpdateMetaFields(key string, cols map[string]interface{}) error {
	return s.db.Model(&models.Meta{}).Where("key = ?", key).Updates(cols).Error
}

// QueryMeta returns only meta values for given key and condition
func (s *MetaService) QueryMeta(key string, condition string) (*[]string, error) {
	var metas []models.Meta
	err := s.db.Where(fmt.Sprintf("%s = ?", key), condition).Find(&metas).Error
	if err != nil {
		return nil, err
	}
	var result []string
	for _, v := range metas {
		result = append(result, v.Value)
	}
	return &result, nil
}

// QueryAllMeta returns all metas for given key and condition
func (s *MetaService) QueryAllMeta(key string, condition string) (*[]models.Meta, error) {
	var metas []models.Meta
	err := s.db.Where(fmt.Sprintf("%s = ?", key), condition).Find(&metas).Error
	if err != nil {
		return nil, err
	}
	return &metas, nil
}
