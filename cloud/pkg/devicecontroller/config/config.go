package config

import (
	"sync"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

var Config Configure
var once sync.Once

// Configure holds the configuration for the devicecontroller module.
type Configure struct {
	v1alpha1.DeviceController
}

// InitConfigure initializes the global Config variable based on the provided
// DeviceController configuration. It is safe to call multiple times.
func InitConfigure(dc *v1alpha1.DeviceController) {
	once.Do(func() {
		Config = Configure{
			DeviceController: *dc,
		}
	})
}
