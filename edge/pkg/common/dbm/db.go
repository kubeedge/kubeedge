package dbm

import (
	"sync"

	"github.com/astaxie/beego/orm"
	"k8s.io/klog"

	//Blank import to run only the init function
	_ "github.com/mattn/go-sqlite3"
)

//DBAccess is Ormer object interface for all transaction processing and switching database
var DBAccess orm.Ormer
var onceDB sync.Once

//InitDBManager initialises the database by syncing the database schema and creating orm
func InitDBManager() {
	// TODO will be deleted after component api featur @kadisi
	InitConfigure()
	InitDB(Get().DriverName, Get().DBName, Get().DataSource)
}

// TODO Ref to component api config @kadisi
func InitDB(driverName, dbName, dataSource string) {
	onceDB.Do(func() {
		// TODO need add switch case @kadisi
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
	})
}
