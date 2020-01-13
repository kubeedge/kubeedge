package dbm

import (
	"github.com/astaxie/beego/orm"
	//Blank import to run only the init function
	_ "github.com/mattn/go-sqlite3"
	"k8s.io/klog"

	commonconfig "github.com/kubeedge/kubeedge/edge/pkg/common/config"
)

//DBAccess is Ormer object interface for all transaction processing and switching database
var DBAccess orm.Ormer

// InitDBConfig Init DB info
func InitDBConfig() {

	if err := orm.RegisterDriver(commonconfig.Get().DriverName, orm.DRSqlite); err != nil {
		klog.Fatalf("Failed to register driver: %v", err)
	}
	if err := orm.RegisterDataBase(
		commonconfig.Get().DBName,
		commonconfig.Get().DriverName,
		commonconfig.Get().DataSource); err != nil {
		klog.Fatalf("Failed to register db: %v", err)
	}
}

//InitDBManager initialises the database by syncing the database schema and creating orm
func InitDBManager() {
	InitDBConfig()
	// sync database schema
	orm.RunSyncdb(commonconfig.Get().DBName, false, true)

	// create orm
	DBAccess = orm.NewOrm()
	DBAccess.Using(commonconfig.Get().DBName)
}
