package config

import (
	"fmt"
	"os"
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
		var errs []error
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

		modulesList := config.CONFIG.GetConfigurationByKey("modules.enabled")
		if modulesList == nil {
			errs = append(errs, fmt.Errorf("get modules.enabled key error"))
			return
		}
		modules := []string{}
		for _, value := range modulesList.([]interface{}) {
			modules = append(modules, value.(string))
		}

		if len(errs) != 0 {
			for _, e := range errs {
				klog.Errorf("%v", e)
			}
			klog.Error("init common config error")
			os.Exit(1)
		}
		c = Configure{
			DriverName: driverName,
			DBName:     dbName,
			DataSource: dataSource,
			Modules:    modules,
		}
		klog.Infof("init common config successfully，config info %++v", c)
	})
}

type Configure struct {
	Modules    []string
	DriverName string
	DBName     string
	DataSource string
}

func Get() *Configure {
	return &c
}
