package util

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
)

func kubeConfig(kubeconfigPath string) (conf *rest.Config, err error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	kubeConfig.QPS = float32(constants.DefaultKubeQPS)
	kubeConfig.Burst = int(constants.DefaultKubeBurst)
	kubeConfig.ContentType = constants.DefaultKubeContentType

	return kubeConfig, nil
}

// KubeClient from config
func KubeClient(kubeConfigPath string) (*kubernetes.Clientset, error) {
	kubeConfig, err := kubeConfig(kubeConfigPath)
	if err != nil {
		klog.Warningf("get kube config failed with error: %s", err)
		return nil, err
	}
	return kubernetes.NewForConfig(kubeConfig)
}

func (co *Common) cleanNameSpace(ns, kubeConfigPath string) error {
	cli, err := KubeClient(kubeConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create KubeClient, error: %s", err)
	}
	return cli.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})
}
