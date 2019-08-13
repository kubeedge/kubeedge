package config

import (
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
)

// Kube container Kubernetes related configuration
var Kube *KubeInfo

// KubeNodeID for the current node
var KubeNodeID string

// KubeNodeName for the current node
var KubeNodeName string

// KubeUpdateNodeFrequency is the time duration for update node status(default is 20s)
var KubeUpdateNodeFrequency time.Duration

//EdgeSiteEnabled is used to enable or disable EdgeSite feature. Default is disabled
var EdgeSiteEnabled bool

func InitKubeConfig() {
	Kube = NewKubeInfo()

	if km, err := config.CONFIG.GetValue("controller.kube.master").ToString(); err != nil {
		klog.Errorf("Controller kube master not set")
	} else {
		Kube.KubeMaster = km
	}
	klog.Infof("Controller kube master: %s", Kube.KubeMaster)

	if kc, err := config.CONFIG.GetValue("controller.kube.kubeconfig").ToString(); err != nil {
		klog.Errorf("Controller kube config not set")
	} else {
		Kube.KubeConfig = kc
	}
	klog.Infof("Controller kube config: %s", Kube.KubeConfig)

	if kn, err := config.CONFIG.GetValue("controller.kube.namespace").ToString(); err == nil {
		Kube.KubeNamespace = kn
	}
	klog.Infof("Controller kube namespace: %s", Kube.KubeNamespace)

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
	if id, err := config.CONFIG.GetValue("controller.kube.node-id").ToString(); err != nil {
		KubeNodeID = ""
	} else {
		KubeNodeID = id
	}
	klog.Infof("Controller kube Node ID: %s", KubeNodeID)

	if name, err := config.CONFIG.GetValue("controller.kube.node-name").ToString(); err != nil {
		KubeNodeName = ""
	} else {
		KubeNodeName = name
	}
	klog.Infof("Controller kube Node Name: %s", KubeNodeName)

	if es, err := config.CONFIG.GetValue("metamanager.edgesite").ToBool(); err != nil {
		EdgeSiteEnabled = false
	} else {
		EdgeSiteEnabled = es
	}
	klog.Infof(" EdgeSite is %t ", EdgeSiteEnabled)
}
