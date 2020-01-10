package utils

import (
	"k8s.io/klog"

	deviceClientSet "github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned"
)

// NewDeviceClient is used to create a deviceClient
func NewDeviceClient() (deviceClientSet.Interface, error) {
	kubeConfig, err := KubeConfig()
	if err != nil {
		klog.Warningf("Get kube config failed with error: %s", err)
		return nil, err
	}
	kubeConfig.ContentConfig.ContentType = "application/json"
	return deviceClientSet.NewForConfig(kubeConfig)
}
