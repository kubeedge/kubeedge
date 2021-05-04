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
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
	"github.com/kubeedge/kubeedge/tests/stubs/common/types"
)

const (
	// Modules
	ControllerHubURL = "http://127.0.0.1:54321"
)

func main() {
	var pod types.FakePod

	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

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
		klog.Errorf("Unmarshal HTTP Response has failed: %v", err)
	}

	resp, err := SendHTTPRequest(http.MethodPost,
		ControllerHubURL+constants.PodResource,
		bytes.NewBuffer(reqBody))
	if err != nil {
		klog.Errorf("Frame HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("HTTP Response reading has failed: %v", err)
	}

	klog.V(4).Infof("AddPod response: %v", contents)
}

// DeletePod deletes a fake pod
func DeletePod(pod types.FakePod) {
	resp, err := SendHTTPRequest(http.MethodDelete,
		ControllerHubURL+constants.PodResource+
			"?name="+pod.Name+"&namespace="+pod.Namespace+"&nodename="+pod.NodeName,
		nil)
	if err != nil {
		klog.Errorf("Frame HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("HTTP Response reading has failed: %v", err)
	}

	klog.V(4).Infof("DeletePod response: %v", contents)
}

// ListPods lists all pods
func ListPods() {
	resp, err := SendHTTPRequest(http.MethodGet, ControllerHubURL+constants.PodResource, nil)
	if err != nil {
		klog.Errorf("Frame HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("HTTP Response reading has failed: %v", err)
	}

	pods := []types.FakePod{}
	err = json.Unmarshal(contents, &pods)
	if err != nil {
		klog.Errorf("Unmarshal message content with error: %s", err)
	}

	klog.V(4).Infof("ListPods result: %v", pods)
}

// SendHTTPRequest launches a http request
func SendHTTPRequest(method, reqAPI string, body io.Reader) (*http.Response, error) {
	var resp *http.Response
	client := &http.Client{}
	req, err := http.NewRequest(method, reqAPI, body)
	if err != nil {
		klog.Errorf("Frame HTTP request failed: %v", err)
		return resp, err
	}
	req.Header.Set("Content-Type", "application/json")
	t := time.Now()
	resp, err = client.Do(req)
	klog.V(4).Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Since(t))
	if err != nil {
		klog.Errorf("HTTP request is failed :%v", err)
		return resp, err
	}
	return resp, nil
}
