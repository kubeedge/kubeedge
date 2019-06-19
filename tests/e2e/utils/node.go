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
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
)

func getpwd() string {
	_, file, _, _ := runtime.Caller(0)
	dir, err := filepath.Abs(filepath.Dir(file))
	if err != nil {
		glog.Errorf("get current dir fail %+v", err)
		return " "
	}
	return dir
}

//DeRegisterNodeFromMaster function to deregister the node from master
func DeRegisterNodeFromMaster(nodehandler, nodename string) error {
	err, resp := SendHttpRequest(http.MethodDelete, nodehandler+"/"+nodename)
	if err != nil {
		Failf("Sending SenHttpRequest failed: %v", err)
		return err
	}
	defer resp.Body.Close()
	Expect(resp.StatusCode).Should(Equal(http.StatusOK))

	return nil
}

//GenerateNodeReqBody function to generate the node request body
func GenerateNodeReqBody(nodeid, nodeselector string) (error, map[string]interface{}) {
	var temp map[string]interface{}

	body := fmt.Sprintf(`{"kind": "Node","apiVersion": "v1","metadata": {"name": "%s","labels": {"name": "edgenode", "disktype":"%s", "node-role.kubernetes.io/edge": ""}}}`, nodeid, nodeselector)
	err := json.Unmarshal([]byte(body), &temp)
	if err != nil {
		Failf("Unmarshal body failed: %v", err)
		return err, temp
	}

	return nil, temp
}

//RegisterNodeToMaster function to register node to master
func RegisterNodeToMaster(UID, nodehandler, nodeselector string) error {
	err, body := GenerateNodeReqBody(UID, nodeselector)
	if err != nil {
		Failf("Unmarshal body failed: %v", err)
		return err
	}

	client := &http.Client{}
	t := time.Now()
	nodebody, err := json.Marshal(body)
	if err != nil {
		Failf("Marshal body failed: %v", err)
		return err
	}
	BodyBuf := bytes.NewReader(nodebody)
	req, err := http.NewRequest(http.MethodPost, nodehandler, BodyBuf)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		Failf("Frame HTTP request failed: %v", err)
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		Failf("Sending HTTP request failed: %v", err)
		return err
	}
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	defer resp.Body.Close()

	Expect(resp.StatusCode).Should(Equal(http.StatusCreated))
	return nil
}

//CheckNodeReadyStatus function to get node status
func CheckNodeReadyStatus(nodehandler, nodename string) string {
	var node v1.Node
	var nodeStatus = "unknown"
	err, resp := SendHttpRequest(http.MethodGet, nodehandler+"/"+nodename)
	if err != nil {
		Failf("Sending SenHttpRequest failed: %v", err)
		return nodeStatus
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Failf("HTTP Response reading has failed: %v", err)
		return nodeStatus
	}
	err = json.Unmarshal(contents, &node)
	if err != nil {
		Failf("Unmarshal HTTP Response has failed: %v", err)
		return nodeStatus
	}

	return string(node.Status.Phase)
}

//CheckNodeDeleteStatus function to check node delete status
func CheckNodeDeleteStatus(nodehandler, nodename string) int {
	err, resp := SendHttpRequest(http.MethodGet, nodehandler+"/"+nodename)
	if err != nil {
		Failf("Sending SenHttpRequest failed: %v", err)
		return -1
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

//HandleConfigmap function to create configmaps for respective edgenodes
func HandleConfigmap(configName chan error, operation, confighandler string, IsEdgeCore bool) {
	var req *http.Request
	var file string
	curpath := getpwd()
	if IsEdgeCore == true {
		file = path.Join(curpath, "../../performance/assets/02-edgeconfigmap.yaml")
	} else {
		file = path.Join(curpath, "../../performance/assets/01-configmap.yaml")
	}
	body, err := ioutil.ReadFile(file)
	if err == nil {
		client := &http.Client{}
		t := time.Now()
		if operation == http.MethodPost {
			BodyBuf := bytes.NewReader(body)
			req, err = http.NewRequest(operation, confighandler, BodyBuf)
			Expect(err).Should(BeNil())
			req.Header.Set("Content-Type", "application/yaml")
		} else if operation == http.MethodPatch {
			jsondata, err := yaml.YAMLToJSON(body)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				return
			}
			BodyBuf := bytes.NewReader(jsondata)
			req, err = http.NewRequest(operation, confighandler, BodyBuf)
			Expect(err).Should(BeNil())
			req.Header.Set("Content-Type", "application/strategic-merge-patch+json")
		} else {
			req, err = http.NewRequest(operation, confighandler, bytes.NewReader([]byte("")))
			Expect(err).Should(BeNil())
			req.Header.Set("Content-Type", "application/json")
		}

		if err != nil {
			Failf("Frame HTTP request failed: %v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			Failf("Sending HTTP request failed: %v", err)
		}
		InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
		defer resp.Body.Close()
		if operation == http.MethodPost {
			Expect(resp.StatusCode).Should(Equal(http.StatusCreated))
		} else {
			Expect(resp.StatusCode).Should(Equal(http.StatusOK))
		}
		configName <- nil

	} else {
		configName <- err
	}
}

//GetConfigmap function to get configmaps for respective edgenodes
func GetConfigmap(apiConfigMap string) (int, []byte) {
	err, resp := SendHttpRequest(http.MethodGet, apiConfigMap)
	if err != nil {
		Failf("Sending SenHttpRequest failed: %v", err)
		return -1,nil
	}
	body,_:=ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	return resp.StatusCode,body

}

//DeleteConfigmap function to delete configmaps
func DeleteConfigmap(apiConfigMap string) int {
	err, resp := SendHttpRequest(http.MethodDelete, apiConfigMap)
	if err != nil {
		Failf("Sending SenHttpRequest failed: %v", err)
		return -1
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TaintEdgeDeployedNode(toTaint bool, taintHandler string) error {
	var temp map[string]interface{}
	var body string
	if toTaint == true {
		body = fmt.Sprintf(`{"spec":{"taints":[{"effect":"NoSchedule","key":"key","value":"value"}]}}`)
	} else {
		body = fmt.Sprintf(`{"spec":{"taints":null}}`)
	}
	err := json.Unmarshal([]byte(body), &temp)
	if err != nil {
		Failf("Unmarshal body failed: %v", err)
		return nil
	}
	nodebody, err := json.Marshal(temp)
	if err != nil {
		Failf("Marshal body failed: %v", err)
		return err
	}
	BodyBuf := bytes.NewReader(nodebody)
	req, err := http.NewRequest(http.MethodPatch, taintHandler, BodyBuf)
	Expect(err).Should(BeNil())
	client := &http.Client{}
	t := time.Now()
	req.Header.Set("Content-Type", "application/strategic-merge-patch+json")
	resp, err := client.Do(req)
	if err != nil {
		Failf("Sending HTTP request failed: %v", err)
		return err
	}
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	defer resp.Body.Close()
	Expect(resp.StatusCode).Should(Equal(http.StatusOK))
	return nil
}

//GetNodes function to get configmaps for respective edgenodes
func GetNodes(api string) v1.NodeList {
	var nodes v1.NodeList
	err, resp := SendHttpRequest(http.MethodGet, api)
	if err != nil {
		Failf("Sending SenHttpRequest failed: %v", err)
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Failf("HTTP Response reading has failed: %v", err)
	}
	err = json.Unmarshal(contents, &nodes)
	if err != nil {
		Failf("Unmarshal HTTP Response has failed: %v", err)
	}

	return nodes
}

func ApplyLabelToNode(apiserver, key, val string) error {
	var temp map[string]interface{}
	var body string
	body = fmt.Sprintf(`{"metadata":{"labels":{"%s":"%s"}}}`, key, val)
	err := json.Unmarshal([]byte(body), &temp)
	if err != nil {
		Failf("Unmarshal body failed: %v", err)
		return nil
	}
	nodebody, err := json.Marshal(temp)
	if err != nil {
		Failf("Marshal body failed: %v", err)
		return err
	}
	BodyBuf := bytes.NewReader(nodebody)
	req, err := http.NewRequest(http.MethodPatch, apiserver, BodyBuf)
	Expect(err).Should(BeNil())
	client := &http.Client{}
	t := time.Now()
	req.Header.Set("Content-Type", "application/strategic-merge-patch+json")
	resp, err := client.Do(req)
	if err != nil {
		Failf("Sending HTTP request failed: %v", err)
		return err
	}
	InfoV6("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	defer resp.Body.Close()
	Expect(resp.StatusCode).Should(Equal(http.StatusOK))
	return nil
}
