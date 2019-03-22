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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"time"

	metav1 "k8s.io/api/apps/v1"
)

//function to get the deployments list
func GetDeployments(list *metav1.DeploymentList, getDeploymentApi string) error {

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

//fucntion to generate deployment body
func GenerateDeploymentBody(deploymentName, image string, port, replica int) (error, map[string]interface{}) {
	var temp map[string]interface{}

	//containerName := "app-" + utils.GetRandomString(5)
	Body := fmt.Sprintf(`{"apiVersion": "apps/v1","kind": "Deployment","metadata": {"name": "%s","labels": {"app": "nginx"}},
				"spec": {"replicas": %v,"selector": {"matchLabels": {"app": "nginx"}},"template": {"metadata": {"labels": {"app": "nginx"}},
				"spec": {"containers": [{"name": "nginx","image": "%s"}]}}}}`, deploymentName, replica, image)
	err := json.Unmarshal([]byte(Body), &temp)
	if err != nil {
		Failf("Unmarshal body failed: %v", err)
		return err, temp
	}

	return nil, temp
}

//Function to handle app deployment/delete deployment.
func HandleDeployment(operation string, apiserver string, UID string, ImageUrl string, replica int) bool {
	var req *http.Request
	var err error
	var body io.Reader

	client := &http.Client{}
	switch operation {
	case "POST":
		Port := RandomInt(10, 100)
		err, body := GenerateDeploymentBody(UID, ImageUrl, Port, replica)
		if err != nil {
			Failf("GenerateDeploymentBody marshalling failed: %v", err)
		}
		respbytes, err := json.Marshal(body)
		if err != nil {
			Failf("Marshalling body failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPost, apiserver, bytes.NewBuffer(respbytes))
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
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	if err != nil {
		// handle error
		Failf("HTTP request is failed :%v", err)
		return false
	}
	return true
}

//function to delete deployment
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

//fucntion to show the os command injuction in combined format
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
