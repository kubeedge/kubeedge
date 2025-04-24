package dao

import (
	"log"
	"sync"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

var dbInstance *gorm.DB
var once sync.Once

func Init(dataSource string, modules ...interface{}) {
	once.Do(func() {
		var err error
		dbInstance, err = gorm.Open(sqlite.Open(dataSource), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to DB: %v", err)
		}
	})

	// Migrate tables for enabled modules
	migrateTables(modules...)
}

// MigrateTables only migrates tables for enabled modules
func migrateTables(modules ...interface{}) {
	for _, m := range modules {
		switch module := m.(type) {
		case *v1alpha2.DeviceTwin:
			if !module.Enable {
				klog.Info("DeviceTwin module is disabled, skipping DB migration")
				continue
			}
			klog.Info("Migrating DB tables for DeviceTwin module")
			if err := dbInstance.AutoMigrate(
				&models.Device{},
				&models.DeviceAttr{},
				&models.DeviceTwin{},
			); err != nil {
				klog.Fatalf("Failed to migrate DeviceTwin tables: %v", err)
			}

		case *v1alpha2.EventBus:
			if !module.Enable {
				klog.Info("EventBus module is disabled, skipping DB migration")
				continue
			}
			klog.Info("Migrating DB tables for EventBus module")
			if err := dbInstance.AutoMigrate(
				&models.SubTopics{},
			); err != nil {
				klog.Fatalf("Failed to migrate EventBus tables: %v", err)
			}

		case *v1alpha2.MetaManager:
			if !module.Enable {
				klog.Info("MetaManager module is disabled, skipping DB migration")
				continue
			}
			klog.Info("Migrating DB tables for MetaManager module")
			if err := dbInstance.AutoMigrate(
				&models.Meta{},
				&models.MetaV2{},
			); err != nil {
				klog.Fatalf("Failed to migrate MetaManager tables: %v", err)
			}

		case *v1alpha2.ServiceBus:
			if !module.Enable {
				klog.Info("ServiceBus module is disabled, skipping DB migration")
				continue
			}
			klog.Info("Migrating DB tables for ServiceBus module")
			if err := dbInstance.AutoMigrate(
				&models.TargetUrls{},
			); err != nil {
				klog.Fatalf("Failed to migrate ServiceBus tables: %v", err)
			}

		default:
			klog.Warningf("Unknown module type: %T", m)
		}
	}
}

func GetDB() *gorm.DB {
	return dbInstance
}
