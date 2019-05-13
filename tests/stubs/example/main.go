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

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
	"github.com/kubeedge/kubeedge/tests/stubs/common/types"
)

const (
	// Modules
	ControllerHubURL = "http://127.0.0.1:54321"
)

func main() {
	var pod types.FakePod
	pod.Name = "TestPod"
	pod.Namespace = constants.NamespaceDefault
	pod.NodeName = "edgenode1"
	pod.Status = constants.PodPending

	AddPod(pod)
	ListPods()

	time.Sleep(10 * time.Second)
	ListPods()

	DeletePod(pod)

	time.Sleep(10 * time.Second)
	ListPods()
}

// AddPod adds a fake pod
func AddPod(pod types.FakePod) {
	reqBody, err := json.Marshal(pod)
	if err != nil {
		log.LOGGER.Errorf("Unmarshal HTTP Response has failed: %v", err)
	}

	err, resp := SendHttpRequest(http.MethodPost,
		ControllerHubURL+constants.PodResource,
		bytes.NewBuffer(reqBody))
	if err != nil {
		log.LOGGER.Errorf("Frame HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.LOGGER.Errorf("HTTP Response reading has failed: %v", err)
	}

	log.LOGGER.Debugf("AddPod response: %v", contents)
}

// DeletePod deletes a fake pod
func DeletePod(pod types.FakePod) {
	err, resp := SendHttpRequest(http.MethodDelete,
		ControllerHubURL+constants.PodResource+
			"?name="+pod.Name+"&namespace="+pod.Namespace+"&nodename="+pod.NodeName,
		nil)
	if err != nil {
		log.LOGGER.Errorf("Frame HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.LOGGER.Errorf("HTTP Response reading has failed: %v", err)
	}

	log.LOGGER.Debugf("DeletePod response: %v", contents)
}

// ListPods lists all pods
func ListPods() {
	err, resp := SendHttpRequest(http.MethodGet, ControllerHubURL+constants.PodResource, nil)
	if err != nil {
		log.LOGGER.Errorf("Frame HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.LOGGER.Errorf("HTTP Response reading has failed: %v", err)
	}

	pods := []types.FakePod{}
	err = json.Unmarshal(contents, &pods)
	if err != nil {
		log.LOGGER.Errorf("Unmarshal message content with error: %s", err)
	}

	log.LOGGER.Debugf("ListPods result: %v", pods)
}

// SendHttpRequest launches a http request
func SendHttpRequest(method, reqApi string, body io.Reader) (error, *http.Response) {
	var resp *http.Response
	client := &http.Client{}
	req, err := http.NewRequest(method, reqApi, body)
	if err != nil {
		log.LOGGER.Errorf("Frame HTTP request failed: %v", err)
		return err, resp
	}
	req.Header.Set("Content-Type", "application/json")
	t := time.Now()
	resp, err = client.Do(req)
	log.LOGGER.Debugf("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	if err != nil {
		log.LOGGER.Errorf("HTTP request is failed :%v", err)
		return err, resp
	}
	return nil, resp
}
