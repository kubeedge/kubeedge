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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	podLabelSelector = "?fieldSelector=spec.nodeName="
)

//GetPods function to get the pods from Edged
func GetPods(apiserver, label string) (v1.PodList, error) {
	var pods v1.PodList
	var resp *http.Response
	var err error

	if len(label) > 0 {
		resp, err = SendHTTPRequest(http.MethodGet, apiserver+podLabelSelector+label)
	} else {
		resp, err = SendHTTPRequest(http.MethodGet, apiserver)
	}
	if err != nil {
		Fatalf("Frame HTTP request failed: %v", err)
		return pods, nil
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Fatalf("HTTP Response reading has failed: %v", err)
		return pods, nil
	}
	err = json.Unmarshal(contents, &pods)
	if err != nil {
		Fatalf("Unmarshal HTTP Response has failed: %v", err)
		return pods, nil
	}
	return pods, nil
}

//GetPodState function to get the pod status and response code
func GetPodState(apiserver string) (string, int) {
	var pod v1.Pod

	resp, err := SendHTTPRequest(http.MethodGet, apiserver)
	if err != nil {
		Fatalf("GetPodState :SenHttpRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			Fatalf("HTTP Response reading has failed: %v", err)
		}
		err = json.Unmarshal(contents, &pod)
		if err != nil {
			Fatalf("Unmarshal HTTP Response has failed: %v", err)
		}
		return string(pod.Status.Phase), resp.StatusCode
	}

	return "", resp.StatusCode
}

//DeletePods function to get the pod status and response code
func DeletePods(apiserver string) (string, int) {
	var pod v1.Pod
	resp, err := SendHTTPRequest(http.MethodDelete, apiserver)
	if err != nil {
		Fatalf("GetPodState :SenHttpRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			Fatalf("HTTP Response reading has failed: %v", err)
		}
		err = json.Unmarshal(contents, &pod)
		if err != nil {
			Fatalf("Unmarshal HTTP Response has failed: %v", err)
		}
		return string(pod.Status.Phase), resp.StatusCode
	}

	return "", resp.StatusCode
}

//CheckPodRunningState function to check the Pod state
func CheckPodRunningState(apiserver string, podlist v1.PodList) {
	gomega.Eventually(func() int {
		var count int
		for _, pod := range podlist.Items {
			state, _ := GetPodState(apiserver + "/" + pod.Name)
			Infof("PodName: %s PodStatus: %s", pod.Name, state)
			if state == "Running" {
				count++
			}
		}
		return count
	}, "600s", "2s").Should(gomega.Equal(len(podlist.Items)), "Application deployment is Unsuccessful, Pod has not come to Running State")
}

//CheckPodDeleteState function to check the Pod state
func CheckPodDeleteState(apiserver string, podlist v1.PodList) {
	var count int
	//skip the edgecore/cloudcore deployment pods and count only application pods deployed on KubeEdge edgen node
	for _, pod := range podlist.Items {
		if strings.Contains(pod.Name, "deployment-") {
			count++
		}
	}
	podCount := len(podlist.Items) - count
	gomega.Eventually(func() int {
		var count int
		for _, pod := range podlist.Items {
			status, statusCode := GetPodState(apiserver + "/" + pod.Name)
			Infof("PodName: %s status: %s StatusCode: %d", pod.Name, status, statusCode)
			if statusCode == 404 {
				count++
			}
		}
		return count
	}, "600s", "4s").Should(gomega.Equal(podCount), "Delete Application deployment is Unsuccessful, Pods are not deleted within the time")
}

//CheckDeploymentPodDeleteState function to check the Pod state
func CheckDeploymentPodDeleteState(apiserver string, podlist v1.PodList) {
	var count int
	//count the edgecore/cloudcore deployment pods and count only application pods deployed on KubeEdge edgen node
	for _, pod := range podlist.Items {
		if strings.Contains(pod.Name, "deployment-") {
			count++
		}
	}
	//podCount := len(podlist.Items) - count
	gomega.Eventually(func() int {
		var count int
		for _, pod := range podlist.Items {
			status, statusCode := GetPodState(apiserver + "/" + pod.Name)
			Infof("PodName: %s status: %s StatusCode: %d", pod.Name, status, statusCode)
			if statusCode == 404 {
				count++
			}
		}
		return count
	}, "240s", "4s").Should(gomega.Equal(count), "Delete Application deployment is Unsuccessful, Pods are not deleted within the time")
}

// NewKubeClient creates kube client from config
func NewKubeClient(kubeConfigPath string) *kubernetes.Clientset {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		Fatalf("Get kube config failed with error: %v", err)
		return nil
	}
	kubeConfig.QPS = 5
	kubeConfig.Burst = 10
	kubeConfig.ContentType = "application/vnd.kubernetes.protobuf"
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		Fatalf("Get kube client failed with error: %v", err)
		return nil
	}
	return kubeClient
}

// WaitforPodsRunning waits util all pods are in running status or timeout
func WaitforPodsRunning(kubeConfigPath string, podlist v1.PodList, timout time.Duration) {
	if len(podlist.Items) == 0 {
		Fatalf("podlist should not be empty")
	}

	podRunningCount := 0
	for _, pod := range podlist.Items {
		if pod.Status.Phase == v1.PodRunning {
			podRunningCount++
		}
	}
	if podRunningCount == len(podlist.Items) {
		Infof("All pods come into running status")
		return
	}

	// new kube client
	kubeClient := NewKubeClient(kubeConfigPath)
	// define signal
	signal := make(chan struct{})
	// define list watcher
	listWatcher := cache.NewListWatchFromClient(
		kubeClient.CoreV1().RESTClient(),
		"pods",
		v1.NamespaceAll,
		fields.Everything())
	// new controller
	_, controller := cache.NewInformer(
		listWatcher,
		&v1.Pod{},
		time.Second*0,
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
				for i := range podlist.Items {
					// update pod status in podlist
					if podlist.Items[i].Name == p.Name {
						Infof("PodName: %s PodStatus: %s", p.Name, p.Status.Phase)
						podlist.Items[i].Status = p.Status
					}
					// check if the pod is in running status
					if podlist.Items[i].Status.Phase == v1.PodRunning {
						count++
					}
				}
				// send an end signal when all pods are in running status
				if len(podlist.Items) == count {
					signal <- struct{}{}
				}
			},
		},
	)

	// run controoler
	podChan := make(chan struct{})
	go controller.Run(podChan)
	defer close(podChan)

	// wait for a signal or timeout
	select {
	case <-signal:
		Infof("All pods come into running status")
	case <-time.After(timout):
		Fatalf("Wait for pods come into running status timeout: %v", timout)
	}
}
