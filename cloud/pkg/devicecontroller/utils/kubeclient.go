package utils

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

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
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Get().KubeAPIConfig.Master, config.Get().KubeAPIConfig.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = float32(config.Get().KubeAPIConfig.QPS)
	kubeConfig.Burst = int(config.Get().KubeAPIConfig.Burst)
	kubeConfig.ContentType = config.Get().KubeAPIConfig.ContentType

	return kubeConfig, err
}
