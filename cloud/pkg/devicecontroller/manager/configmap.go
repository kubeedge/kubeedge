package manager

import (
	v1 "k8s.io/api/core/v1"
	k8sinformer "k8s.io/client-go/informers"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
)

// ConfigMapManager is a manager for configmap of deviceProfile
type ConfigMapManager struct {
	Informer k8sinformer.SharedInformerFactory
}

// NewConfigMapManager is function to return new ConfigMapManager
func NewConfigMapManager() *ConfigMapManager {
	return &ConfigMapManager{
		Informer: informers.GetInformersManager().GetK8sInformerFactory(),
	}
}

// GetConfigMap gets the configmap using the k8s informer cache
func (m *ConfigMapManager) GetConfigMap(namespace string, name string) (*v1.ConfigMap, error) {
	configMap, err := m.Informer.Core().V1().ConfigMaps().Lister().ConfigMaps(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	// Objects returned by informer must be treated as read-only.
	// So use DeepCopy to create a new ConfigMap.
	return configMap.DeepCopy(), nil
}
