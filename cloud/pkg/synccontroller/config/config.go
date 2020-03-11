package config

import (
	"sync"

	configv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	KubeAPIConfig  *configv1alpha1.KubeAPIConfig
	SyncController *configv1alpha1.SyncController
}

func InitConfigure(sc *configv1alpha1.SyncController, kubeAPIConfig *configv1alpha1.KubeAPIConfig) {
	once.Do(func() {
		Config = Configure{
			KubeAPIConfig:  kubeAPIConfig,
			SyncController: sc,
		}
	})
}
