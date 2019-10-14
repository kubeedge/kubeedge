package utils

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/common/constants"
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
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Conf().Kube.Master, config.Conf().Kube.Kubeconfig)
	if err != nil {
		return nil, err
	}

	kubeConfig.QPS = constants.DefaultKubeQPS
	kubeConfig.Burst = constants.DefaultKubeBurst
	kubeConfig.ContentType = constants.DefaultKubeContentType

	return kubeConfig, err
}
