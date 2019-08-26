package utils

import (
	"github.com/kubeedge/kubeedge/cloud/pkg/admissioncontroller/config"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeClient from config
func KubeClient() (*kubernetes.Clientset, error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Kube.KubeMaster, config.Kube.KubeConfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(kubeConfig)
}
