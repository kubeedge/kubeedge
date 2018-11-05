package dbm

import (
	"strings"

	"edge-core/beehive/pkg/common/config"
	"edge-core/beehive/pkg/common/log"
	"github.com/astaxie/beego/orm"
	_ "github.com/mattn/go-sqlite3"
)

var (
	driverName string
	dbName     string
	dataSource string
)

var DBAccess orm.Ormer

func RegisterModel(moduleName string, m interface{}) {
	if isModuleEnabled(moduleName) {
		orm.RegisterModel(m)
		log.LOGGER.Infof("DB meta for module %s has been registered", moduleName)
	} else {
		log.LOGGER.Infof("DB meta for module %s has not been registered because this module is not enabled", moduleName)
	}
}

func init() {
	//Init DB info
	driverName, _ = config.CONFIG.GetValue("database.driver").ToString()
	dbName, _ = config.CONFIG.GetValue("database.name").ToString()
	dataSource, _ = config.CONFIG.GetValue("database.source").ToString()
	if driverName == "" {
		driverName = "sqlite3"
	}
	if dbName == "" {
		dbName = "default"
	}
	if dataSource == "" {
		dataSource = "edge.db"
	}

	orm.RegisterDriver(driverName, orm.DRSqlite)
	orm.RegisterDataBase(dbName, driverName, dataSource)
}

func InitDBManager() {
	// sync database schema
	orm.RunSyncdb(dbName, false, true)

	// create orm
	DBAccess = orm.NewOrm()
	DBAccess.Using(dbName)
}

func isModuleEnabled(m string) bool {
	modules := config.CONFIG.GetConfigurationByKey("modules.enabled")
	if modules != nil {
		for _, value := range modules.([]interface{}) {
			if m == value.(string) {
				return true
			}
		}
	}
	return false
}

// IsNonUniqueNameError tests if the error returned by sqlite is unique.
// It will check various sqlite versions.
func IsNonUniqueNameError(err error) bool {
	str := err.Error()
	if strings.HasSuffix(str, "are not unique") || strings.Contains(str, "UNIQUE constraint failed") || strings.HasSuffix(str, "constraint failed") {
		return true
	}
	return false
}
