/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"context"
	"time"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	edgeclientset "github.com/kubeedge/api/client/clientset/versioned"
)

func GetPods(c clientset.Interface, ns string, labelSelector labels.Selector, fieldSelector fields.Selector) (*v1.PodList, error) {
	options := metav1.ListOptions{}

	if fieldSelector != nil {
		options.FieldSelector = fieldSelector.String()
	}

	if labelSelector != nil {
		options.LabelSelector = labelSelector.String()
	}

	return c.CoreV1().Pods(ns).List(context.TODO(), options)
}

func GetPod(c clientset.Interface, ns, name string) (*v1.Pod, error) {
	return c.CoreV1().Pods(ns).Get(context.TODO(), name, metav1.GetOptions{})
}

func DeletePod(c clientset.Interface, ns, name string) error {
	return c.CoreV1().Pods(ns).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func CreatePod(c clientset.Interface, pod *v1.Pod) (*v1.Pod, error) {
	return c.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
}

func WaitForPodsToDisappear(c clientset.Interface, ns string, label labels.Selector, interval, timeout time.Duration) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		Infof("Waiting for pod with label %s to disappear", label.String())
		options := metav1.ListOptions{LabelSelector: label.String()}
		pods, err := c.CoreV1().Pods(ns).List(context.TODO(), options)
		if err != nil {
			return false, err
		}

		if pods != nil && len(pods.Items) == 0 {
			Infof("Pod with label %s no longer exists", label.String())
			return true, nil
		}

		return false, nil
	})
}

// CheckPodDeleteState check whether the given pod list is deleted successfully
func CheckPodDeleteState(c clientset.Interface, podList *v1.PodList) {
	podCount := len(podList.Items)

	errInfo := "Pods of deploy are not deleted within the time"

	gomega.Eventually(func() int {
		var count int
		for _, pod := range podList.Items {
			_, err := GetPod(c, pod.Namespace, pod.Name)
			if err != nil && apierrors.IsNotFound(err) {
				count++
				continue
			}

			if err != nil {
				klog.Errorf("get pod %s/%s error", pod.Namespace, pod.Name)
				continue
			}

			Infof("Pod %s/%s still exist", pod.Namespace, pod.Name)
		}

		return count
	}, "240s", "4s").Should(gomega.Equal(podCount), errInfo)
}

// NewKubeClient creates kube client from config
func NewKubeClient(kubeConfigPath string) clientset.Interface {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		Fatalf("Get kube config failed with error: %v", err)
		return nil
	}
	kubeConfig.QPS = 5
	kubeConfig.Burst = 10
	kubeConfig.ContentType = "application/vnd.kubernetes.protobuf"
	kubeClient, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		Fatalf("Get kube client failed with error: %v", err)
		return nil
	}
	return kubeClient
}

// NewKubeEdgeClient creates kubeEdge CRD client from config
func NewKubeEdgeClient(kubeConfigPath string) edgeclientset.Interface {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		Fatalf("Get kube config failed with error: %v", err)
		return nil
	}
	kubeConfig.QPS = 5
	kubeConfig.Burst = 10
	edgeClientSet, err := edgeclientset.NewForConfig(kubeConfig)
	if err != nil {
		Fatalf("Get kubeEdge client failed with error: %v", err)
		return nil
	}
	return edgeClientSet
}

// WaitForPodsRunning waits util all pods are in running status or timeout
func WaitForPodsRunning(c clientset.Interface, podList *v1.PodList, timeout time.Duration) {
	if len(podList.Items) == 0 {
		Fatalf("podList should not be empty")
	}

	gomega.Eventually(func() int {
		var count int
		for i := range podList.Items {
			pod := &podList.Items[i]
			current, err := GetPod(c, pod.Namespace, pod.Name)
			if err != nil {
				Errorf("get pod %s/%s error: %v", pod.Namespace, pod.Name, err)
				continue
			}

			pod.Status = current.Status
			if pod.Status.Phase == v1.PodRunning {
				count++
				continue
			}

			Infof("Pod %s/%s is still %s", pod.Namespace, pod.Name, pod.Status.Phase)
		}

		return count
	}, timeout, 4*time.Second).Should(gomega.Equal(len(podList.Items)), "not all pods reached the Running phase")

	Infof("All pods come into running status")
}
