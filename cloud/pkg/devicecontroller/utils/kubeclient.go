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
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.KubeMaster, config.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = config.KubeQPS
	kubeConfig.Burst = config.KubeBurst
	kubeConfig.ContentType = config.KubeContentType

	return kubeConfig, err
}
