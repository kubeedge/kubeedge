package config

import (
	"sync"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

var Config Configure
var once sync.Once

// Configure holds the configuration for the router module.
type Configure struct {
	v1alpha1.Router
}

// InitConfigure initializes the global Config variable based on the provided
// Router configuration. It is safe to call multiple times (only executes once).
func InitConfigure(router *v1alpha1.Router) {
	once.Do(func() {
		Config = Configure{
			Router: *router,
		}
	})
}
