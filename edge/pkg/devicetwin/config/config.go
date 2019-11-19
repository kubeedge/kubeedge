package config

import (
	"fmt"
	"os"
	"sync"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
)

var c Configure
var once sync.Once

type Configure struct {
	NodeID string
}

func InitConfigure() {
	once.Do(func() {
		var errs []error
		defer func() {
			if len(errs) != 0 {
				for _, e := range errs {
					klog.Errorf("%v", e)
				}
				klog.Error("init devicetwin config error")
				os.Exit(1)
			} else {
				klog.Infof("init devicetwin config successfully，config info %++v", c)
			}
		}()
		nodeID, err := config.CONFIG.GetValue("edgehub.controller.node-id").ToString()
		if err != nil {
			errs = append(errs, fmt.Errorf("get edgehub.controller.node-id key error %v"), err)
		}
		c = Configure{
			NodeID: nodeID,
		}
	})
}

func Get() Configure {
	return c
}
