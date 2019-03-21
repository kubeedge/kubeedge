package dbm

import (
	"os"
	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	//Blank import to run only the init function
	_ "github.com/mattn/go-sqlite3"
)

const (
	// defaultDriverName is sqlite3
	defaultDriverName = "sqlite3"
	// defaultDbName is default
	defaultDbName = "default"
	// defaultDataSource is edge.db
	defaultDataSource = "edge.db"
)

var (
	driverName string
	dbName     string
	dataSource string
)

//DBAccess is Ormer object interface for all transaction processing and switching database
var DBAccess orm.Ormer

//RegisterModel registers the defined model in the orm if model is enabled
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
		driverName = defaultDriverName
	}
	if dbName == "" {
		dbName = defaultDbName
	}
	if dataSource == "" {
		dataSource = defaultDataSource
	}

	if err := orm.RegisterDriver(driverName, orm.DRSqlite); err != nil {
		log.LOGGER.Fatalf("Failed to register driver: %v", err)
	}
	if err := orm.RegisterDataBase(dbName, driverName, dataSource); err != nil {
		log.LOGGER.Fatalf("Failed to register db: %v", err)
	}
}

//InitDBManager initialises the database by syncing the database schema and creating orm
func InitDBManager() {
	// sync database schema
	orm.RunSyncdb(dbName, false, true)

	// create orm
	DBAccess = orm.NewOrm()
	DBAccess.Using(dbName)
}

// Cleanup cleans up resources
func Cleanup() {
	cleanDBFile(dataSource)
}

// cleanDBFile removes db file
func cleanDBFile(fileName string) {
	// Remove db file
	err := os.Remove(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			log.LOGGER.Infof("DB file %s is not existing", fileName)
		} else {
			log.LOGGER.Errorf("Failed to remove DB file %s: %v", fileName, err)
		}
	}
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
