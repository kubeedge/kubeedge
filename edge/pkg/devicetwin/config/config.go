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

		nodeID, err := config.CONFIG.GetValue("edgehub.controller.node-id").ToString()
		if err != nil {
			errs = append(errs, fmt.Errorf("get edgehub.controller.node-id key error %v", err))
		}

		if len(errs) != 0 {
			for _, e := range errs {
				klog.Errorf("%v", e)
			}
			klog.Error("init devicetwin config error")
			os.Exit(1)
		}
		c = Configure{
			NodeID: nodeID,
		}
		klog.Infof("init devicetwin config successfully，config info %++v", c)
	})
}

func Get() *Configure {
	return &c
}
