package config

import (
	"sync"

	configv1alpha1 "github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

var Config Configure
var once sync.Once

// Configure holds the configuration for the synccontroller module.
type Configure struct {
	SyncController *configv1alpha1.SyncController
}

// InitConfigure initializes the global Config variable based on the provided
// SyncController configuration. It is safe to call multiple times.
func InitConfigure(sc *configv1alpha1.SyncController) {
	once.Do(func() {
		Config = Configure{
			SyncController: sc,
		}
	})
}
