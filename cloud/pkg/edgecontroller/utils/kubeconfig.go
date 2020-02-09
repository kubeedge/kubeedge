package utils

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
)

// KubeConfig from flags
func KubeConfig() (conf *rest.Config, err error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Config.KubeAPIConfig.Master,
		config.Config.KubeAPIConfig.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = float32(config.Config.KubeAPIConfig.QPS)
	kubeConfig.Burst = int(config.Config.KubeAPIConfig.Burst)
	kubeConfig.ContentType = config.Config.KubeAPIConfig.ContentType

	return kubeConfig, nil
}
