package utils

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
)

// KubeConfig from flags
func KubeConfig() (conf *rest.Config, err error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Get().KubeAPIConfig.Master,
		config.Get().KubeAPIConfig.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = float32(config.Get().KubeAPIConfig.QPS)
	kubeConfig.Burst = int(config.Get().KubeAPIConfig.Burst)
	kubeConfig.ContentType = config.Get().KubeAPIConfig.ContentType

	return kubeConfig, nil
}
