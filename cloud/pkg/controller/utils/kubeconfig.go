package utils

import (
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/config"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeConfig from flags
func KubeConfig() (conf *rest.Config, err error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Kube.KubeMaster, config.Kube.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = config.Kube.KubeQPS
	kubeConfig.Burst = config.Kube.KubeBurst
	kubeConfig.ContentType = config.Kube.KubeContentType

	return kubeConfig, err
}
