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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	edgeclientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
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

	podRunningCount := 0
	for _, pod := range podList.Items {
		if pod.Status.Phase == v1.PodRunning {
			podRunningCount++
		}
	}

	if podRunningCount == len(podList.Items) {
		Infof("All pods come into running status")
		return
	}

	// define signal
	signal := make(chan struct{})

	// define list watcher
	listWatcher := cache.NewListWatchFromClient(c.CoreV1().RESTClient(), "pods", v1.NamespaceAll, fields.Everything())

	// new controller
	_, controller := cache.NewInformer(listWatcher, &v1.Pod{}, 0,
		cache.ResourceEventHandlerFuncs{
			// receive update events
			UpdateFunc: func(oldObj, newObj interface{}) {
				// check update obj
				p, ok := newObj.(*v1.Pod)
				if !ok {
					Fatalf("Failed to cast observed object to pod")
				}

				// calculate the pods in running status
				count := 0
				for i := range podList.Items {
					// update pod status in podList
					if podList.Items[i].Name == p.Name {
						Infof("PodName: %s PodStatus: %s", p.Name, p.Status.Phase)
						podList.Items[i].Status = p.Status
					}
					// check if the pod is in running status
					if podList.Items[i].Status.Phase == v1.PodRunning {
						count++
					}
				}

				// send an end signal when all pods are in running status
				if len(podList.Items) == count {
					signal <- struct{}{}
				}
			},
		},
	)

	// run controller
	podChan := make(chan struct{})
	go controller.Run(podChan)
	defer close(podChan)

	// wait for a signal or timeout
	select {
	case <-signal:
		Infof("All pods come into running status")
	case <-time.After(timeout):
		Fatalf("Wait for pods come into running status timeout: %v", timeout)
	}
}
