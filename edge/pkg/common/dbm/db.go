package dbm

import (
	"sync"

	"github.com/astaxie/beego/orm"

	//Blank import to run only the init function
	_ "github.com/mattn/go-sqlite3"
	"k8s.io/klog/v2"
)

// DBAccess is Ormer object interface for all transaction processing and switching database
var DBAccess orm.Ormer
var once sync.Once

// InitDBConfig Init DB info
func InitDBConfig(driverName, dbName, dataSource string) {
	once.Do(func() {
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
		if err := orm.RunSyncdb(dbName, false, true); err != nil {
			klog.Errorf("run sync db error %v", err)
		}
		// create orm
		DBAccess = orm.NewOrm()
		if err := DBAccess.Using(dbName); err != nil {
			klog.Errorf("Using db access error %v", err)
		}
	})
}
