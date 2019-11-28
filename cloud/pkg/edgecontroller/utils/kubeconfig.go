package utils

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
)

// KubeConfig from flags
func KubeConfig() (conf *rest.Config, err error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Get().KubeMaster, config.Get().KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = config.Get().KubeQPS
	kubeConfig.Burst = config.Get().KubeBurst
	kubeConfig.ContentType = config.Get().KubeContentType

	return kubeConfig, nil
}
