package config

import (
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
)

var c Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgeController
	KubeAPIConfig  v1alpha1.KubeAPIConfig
	NodeName       string
	EdgeSiteEnable bool
}

func InitConfigure(controller *v1alpha1.EdgeController, k *v1alpha1.KubeAPIConfig, nodeName string, edgesite bool) {
	once.Do(func() {
		c = Configure{
			EdgeController: *controller,
			KubeAPIConfig:  *k,
			NodeName:       nodeName,
			EdgeSiteEnable: edgesite,
		}
	})
}
func Get() *Configure {
	return &c
}
