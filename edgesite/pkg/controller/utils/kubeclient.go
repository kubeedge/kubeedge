package utils

import (
	"github.com/kubeedge/beehive/pkg/common/log"

	"k8s.io/client-go/kubernetes"
)

// KubeClient from config
func KubeClient() (*kubernetes.Clientset, error) {
	kubeConfig, err := KubeConfig()
	if err != nil {
		log.LOGGER.Warnf("get kube config failed with error: %s", err)
		return nil, err
	}
	return kubernetes.NewForConfig(kubeConfig)
}
