package utils

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/common/constants"
)

// KubeConfig from flags
func KubeConfig() (conf *rest.Config, err error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Conf().Kube.Master, config.Conf().Kube.Kubeconfig)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = constants.DefaultKubeQPS
	kubeConfig.Burst = constants.DefaultKubeBurst
	kubeConfig.ContentType = constants.DefaultKubeContentType
	return kubeConfig, nil
}
