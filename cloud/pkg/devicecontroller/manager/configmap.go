package manager

import (
	"sync"
)

// ConfigMapManager is a manager for configmap of deviceProfile
type ConfigMapManager struct {
	// ConfigMap, key is nodeID, value is configmap
	ConfigMap sync.Map
}

// NewConfigMapManager is function to return new ConfigMapManager
func NewConfigMapManager() *ConfigMapManager {
	return &ConfigMapManager{}
}
