package config

import (
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
)

// Kube container Kubernetes related configuration
var Kube *KubeInfo

//EdgeSiteEnabled is used to enable or disable EdgeSite feature. Default is disabled
var EdgeSiteEnabled bool

func InitKubeConfig() {
	Kube = NewKubeInfo()

	if kct, err := config.CONFIG.GetValue("controller.kube.content_type").ToString(); err == nil {
		Kube.KubeContentType = kct
	}
	klog.Infof("Controller kube content type: %s", Kube.KubeContentType)

	if kqps, err := config.CONFIG.GetValue("controller.kube.qps").ToFloat64(); err == nil {
		Kube.KubeQPS = float32(kqps)
	}
	klog.Infof("Controller kube QPS: %f", Kube.KubeQPS)

	if kb, err := config.CONFIG.GetValue("controller.kube.burst").ToInt(); err == nil {
		Kube.KubeBurst = kb
	}
	klog.Infof("Controller kube burst: %d", Kube.KubeBurst)

	if kuf, err := config.CONFIG.GetValue("controller.kube.node_update_frequency").ToInt64(); err == nil {
		Kube.KubeUpdateNodeFrequency = time.Duration(kuf) * time.Second
	}
	klog.Infof("Controller kube update frequency: %v", Kube.KubeUpdateNodeFrequency)

	if es, err := config.CONFIG.GetValue("metamanager.edgesite").ToBool(); err != nil {
		EdgeSiteEnabled = false
	} else {
		EdgeSiteEnabled = es
	}
	klog.Infof(" EdgeSite is %t ", EdgeSiteEnabled)
}
