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
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"time"

	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/api/core/v1"
)

func newDeployment(name, imgUrl, nodeselector string, replicas int) *apps.Deployment {
	deployment := apps.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels: map[string]string{"app": "nginx"},
		},
		Spec: apps.DeploymentSpec{
			Replicas: func() *int32 { i := int32(replicas); return &i }(),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "nginx"}},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "nginx"},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "nginx",
							Image: imgUrl,
						},
					},
					NodeSelector: map[string]string{"disktype": nodeselector},
				},
			},
		},
	}
	return &deployment
}
func newPodObj(podName, imgUrl, nodeselector string) *v1.Pod {
	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        podName,
			Labels: map[string]string{"app": "nginx"},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "nginx",
					Image: imgUrl,
					Ports:[]v1.ContainerPort{{HostPort: 80, ContainerPort: 80}},
				},
			},
			NodeSelector: map[string]string{"disktype": nodeselector},
		},
	}
	return &pod
}
//GetDeployments to get the deployments list
func GetDeployments(list *apps.DeploymentList, getDeploymentApi string) error {

	err, resp := SendHttpRequest(http.MethodGet, getDeploymentApi)
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Failf("HTTP Response reading has failed: %v", err)
		return err
	}
	err = json.Unmarshal(contents, &list)
	if err != nil {
		Failf("Unmarshal HTTP Response has failed: %v", err)
		return err
	}
	return nil

}
//HandlePod to handle app deployment/delete using pod spec.
func HandlePod(operation string, apiserver string, UID string, ImageUrl, nodeselector string) bool {
	var req *http.Request
	var err error
	var body io.Reader

	client := &http.Client{}
	switch operation {
	case "POST":
		body:= newPodObj(UID, ImageUrl, nodeselector)
		if err != nil {
			Failf("GenerateDeploymentBody marshalling failed: %v", err)
		}
		respBytes, err := json.Marshal(body)
		if err != nil {
			Failf("Marshalling body failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPost, apiserver, bytes.NewBuffer(respBytes))
	case "DELETE":
		req, err = http.NewRequest(http.MethodDelete, apiserver+UID, body)
	}
	if err != nil {
		// handle error
		Failf("Frame HTTP request failed: %v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return false
	}
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	return true
}

//HandleDeployment to handle app deployment/delete deployment.
func HandleDeployment(operation, apiserver, UID, ImageUrl, nodeselector string, replica int) bool {
	var req *http.Request
	var err error
	var body io.Reader

	client := &http.Client{}
	switch operation {
	case "POST":
		//err, body := GenerateDeploymentBody(UID, ImageUrl, nodeselector, replica)
		depObj := newDeployment(UID, ImageUrl, nodeselector, replica)
		if err != nil {
			Failf("GenerateDeploymentBody marshalling failed: %v", err)
		}
		respBytes, err := json.Marshal(depObj)
		if err != nil {
			Failf("Marshalling body failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPost, apiserver, bytes.NewBuffer(respBytes))
	case "DELETE":
		req, err = http.NewRequest(http.MethodDelete, apiserver+UID, body)
	}
	if err != nil {
		// handle error
		Failf("Frame HTTP request failed: %v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return false
	}
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	return true
}

//DeleteDeployment to delete deployment
func DeleteDeployment(DeploymentApi, deploymentname string) int {
	err, resp := SendHttpRequest(http.MethodDelete, DeploymentApi+"/"+deploymentname)
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return -1
	}

	defer resp.Body.Close()

	return resp.StatusCode
}

//PrintCombinedOutput to show the os command injuction in combined format
func PrintCombinedOutput(cmd *exec.Cmd) error {
	Info("===========> Executing: %s\n", strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		Failf("CombinedOutput failed", err)
		return err
	}
	if len(output) > 0 {
		Info("=====> Output: %s\n", string(output))
	}
	return nil
}
