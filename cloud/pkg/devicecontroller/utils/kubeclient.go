package utils

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
)

// KubeClient from config
func KubeClient() (*kubernetes.Clientset, error) {
	kubeConfig, err := KubeConfig()
	if err != nil {
		klog.Warningf("Get kube config failed with error: %s", err)
		return nil, err
	}
	return kubernetes.NewForConfig(kubeConfig)
}

// KubeConfig from flags
func KubeConfig() (conf *rest.Config, err error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Config.KubeAPIConfig.Master, config.Config.KubeAPIConfig.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = float32(config.Config.KubeAPIConfig.QPS)
	kubeConfig.Burst = int(config.Config.KubeAPIConfig.Burst)
	kubeConfig.ContentType = config.Config.KubeAPIConfig.ContentType

	return kubeConfig, err
}
