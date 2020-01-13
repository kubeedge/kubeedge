package dbm

import (
	"github.com/astaxie/beego/orm"
	//Blank import to run only the init function
	_ "github.com/mattn/go-sqlite3"
	"k8s.io/klog"

	commonconfig "github.com/kubeedge/kubeedge/edge/pkg/common/config"
)

// DBAccess is Ormer object interface for all transaction processing and switching database
var DBAccess orm.Ormer

// InitDBManager initialises the database by syncing the database schema and creating orm
func InitDBManager() {
	// TODO will changed at component api config feature @kadisi
	InitDBConfig(commonconfig.Get().DriverName, commonconfig.Get().DBName, commonconfig.Get().DataSource)
}

// InitDBConfig Init DB info
func InitDBConfig(driverName, dbName, dataSource string) {
	if err := orm.RegisterDriver(driverName, orm.DRSqlite); err != nil {
		klog.Fatalf("Failed to register driver: %v", err)
	}
	if err := orm.RegisterDataBase(
		dbName,
		driverName,
		dataSource); err != nil {
		klog.Fatalf("Failed to register db: %v", err)
	}
	// sync database schema
	orm.RunSyncdb(dbName, false, true)

	// create orm
	DBAccess = orm.NewOrm()
	DBAccess.Using(dbName)
}
