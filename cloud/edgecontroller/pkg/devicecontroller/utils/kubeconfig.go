package utils

import (
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/config"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

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
