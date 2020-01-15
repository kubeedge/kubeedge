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

func InitConfigure(ec *v1alpha1.EdgeController, kubeAPIConfig *v1alpha1.KubeAPIConfig, nodeName string, edgesite bool) {
	once.Do(func() {
		c = Configure{
			EdgeController: *ec,
			KubeAPIConfig:  *kubeAPIConfig,
			NodeName:       nodeName,
			EdgeSiteEnable: edgesite,
		}
	})
}
func Get() *Configure {
	return &c
}
