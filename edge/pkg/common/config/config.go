package config

import (
	"sync"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
)

const (
	// defaultDriverName is sqlite3
	defaultDriverName = "sqlite3"
	// defaultDbName is default
	defaultDbName = "default"
	// defaultDataSource is edge.db
	defaultDataSource = "edge.db"
)

var c Configure
var once sync.Once

func init() {
	once.Do(func() {
		driverName, err := config.CONFIG.GetValue("database.driver").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			driverName = defaultDriverName
			klog.Infof("can not get database.driver key, use default %v", driverName)
		}
		dbName, err := config.CONFIG.GetValue("database.name").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			dbName = defaultDbName
			klog.Infof("can not get database.name key, use default %v", dbName)
		}
		dataSource, err := config.CONFIG.GetValue("database.source").ToString()
		if err != nil {
			// Guaranteed forward compatibility @kadisi
			dataSource = defaultDataSource
			klog.Infof("can not get database.source key, use default %v", dataSource)
		}

		c = Configure{
			DriverName: driverName,
			DBName:     dbName,
			DataSource: dataSource,
		}
		klog.Infof("init common config successfully，config info %++v", c)
	})
}

type Configure struct {
	DriverName string
	DBName     string
	DataSource string
}

func Get() *Configure {
	return &c
}
