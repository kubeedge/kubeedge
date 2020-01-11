package config

import (
	"os"
	"sync"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
)

var c Configure
var once sync.Once

type Configure struct {
	StrategyName string
}

func InitConfigure() {
	once.Do(func() {
		var errs []error
		if len(errs) != 0 {
			for _, e := range errs {
				klog.Errorf("%v", e)
			}
			klog.Error("init edgemesh config error, exit")
			os.Exit(1)
		}
		strategyName := config.CONFIG.GetConfigurationByKey("mesh.loadbalance.strategy-name").(string)
		c = Configure{
			StrategyName: strategyName,
		}
		klog.Infof("init edgemesh config successfully，config info %++v", c)
	})

}
func Get() *Configure {
	return &c
}
