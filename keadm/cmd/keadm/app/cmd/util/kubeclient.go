package util

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

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
		return nil, fmt.Errorf("get kube config failed with error: %s", err)
	}
	return kubernetes.NewForConfig(kubeConfig)
}

func (co *Common) CleanNameSpace(ns, kubeConfigPath string) error {
	cli, err := KubeClient(kubeConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create KubeClient, error: %s", err)
	}
	return cli.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})
}

// IsCloudcoreContainerRunning judge whether cloudcore pod is running
func IsCloudcoreContainerRunning(ns, kubeConfigPath string) (bool, error) {
	cli, err := KubeClient(kubeConfigPath)
	if err != nil {
		return false, fmt.Errorf("failed to create KubeClient, error: %s", err)
	}
	podList, err := cli.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to query pods, error: %s", err)
	}
	if podList.Items == nil || len(podList.Items) == 0 {
		return false, nil
	}
	return true, nil
}
