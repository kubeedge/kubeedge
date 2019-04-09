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
	"k8s.io/api/core/v1"
	"net/http"

	. "github.com/onsi/gomega"
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
		err, resp = SendHttpRequest(http.MethodGet, apiserver+podLabelSelector+label)
	} else {
		err, resp = SendHttpRequest(http.MethodGet, apiserver)
	}
	if err != nil {
		Failf("Frame HTTP request failed: %v", err)
		return pods, nil
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Failf("HTTP Response reading has failed: %v", err)
		return pods, nil
	}
	err = json.Unmarshal(contents, &pods)
	if err != nil {
		Failf("Unmarshal HTTP Response has failed: %v", err)
		return pods, nil
	}
	return pods, nil
}

//GetPodState function to get the pod status and response code
func GetPodState(apiserver string) (string, int) {
	var pod v1.Pod

	err, resp := SendHttpRequest(http.MethodGet, apiserver)
	if err != nil {
		Failf("GetPodState :SenHttpRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			Failf("HTTP Response reading has failed: %v", err)
		}
		err = json.Unmarshal(contents, &pod)
		if err != nil {
			Failf("Unmarshal HTTP Response has failed: %v", err)
		}
		return string(pod.Status.Phase), resp.StatusCode
	}

	return "", resp.StatusCode
}

//DeletePods function to get the pod status and response code
func DeletePods(apiserver string) (string, int) {
	var pod v1.Pod
	err, resp := SendHttpRequest(http.MethodDelete, apiserver)
	if err != nil {
		Failf("GetPodState :SenHttpRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			Failf("HTTP Response reading has failed: %v", err)
		}
		err = json.Unmarshal(contents, &pod)
		if err != nil {
			Failf("Unmarshal HTTP Response has failed: %v", err)
		}
		return string(pod.Status.Phase), resp.StatusCode
	}

	return "", resp.StatusCode
}

//CheckPodRunningState function to check the Pod state
func CheckPodRunningState(apiserver string, podlist v1.PodList) {
	Eventually(func() int {
		var count int
		for _, pod := range podlist.Items {
			state, _ := GetPodState(apiserver + "/" + pod.Name)
			InfoV2("PodName: %s PodStatus: %s", pod.Name, state)
			if state == "Running" {
				count++
			}
		}
		return count
	}, "240s", "4s").Should(Equal(len(podlist.Items)), "Delete Application deployment is Unsuccessfull, Pod has not come to Running State")

}

//CheckPodDeleteState function to check the Pod state
func CheckPodDeleteState(apiserver string, podlist v1.PodList) {
	Eventually(func() int {
		var count int
		for _, pod := range podlist.Items {
			status, statusCode := GetPodState(apiserver + "/" + pod.Name)
			InfoV2("PodName: %s status: %s StatusCode: %d", pod.Name, status, statusCode)
			if statusCode == 404 {
				count++
			}
		}
		return count
	}, "240s", "4s").Should(Equal(len(podlist.Items)), "Delete Application deployment is Unsuccessfull, Pod has not come to Running State")

}
