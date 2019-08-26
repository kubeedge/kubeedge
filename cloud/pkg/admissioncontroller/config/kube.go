package config

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
)

// Kube container Kubernetes related configuration
var Kube *KubeInfo

func init() {
	Kube = newKubeInfo()

	if km, err := config.CONFIG.GetValue("devicecontroller.kube.master").ToString(); err != nil {
		klog.Error("kube master is not set for devicecontroller")
	} else {
		Kube.KubeMaster = km
	}
	klog.Infof("kube master: %s", Kube.KubeMaster)

	if kc, err := config.CONFIG.GetValue("devicecontroller.kube.kubeconfig").ToString(); err != nil {
		klog.Error("kube config is not set for devicecontroller")
	} else {
		Kube.KubeConfig = kc
	}
	klog.Infof("kube config: %s", Kube.KubeConfig)
}
