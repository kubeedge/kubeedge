package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
	configv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
)

var c Configure
var once sync.Once

type Configure struct {
	KubeAPIConfig  *v1alpha1.KubeAPIConfig
	SyncController *configv1alpha1.SyncController
}

func InitConfigure(sc *configv1alpha1.SyncController, kubeAPIConfig *v1alpha1.KubeAPIConfig) {
	once.Do(func() {
		c = Configure{
			KubeAPIConfig:  kubeAPIConfig,
			SyncController: sc,
		}
	})
}

func Get() *Configure {
	return &c
}
