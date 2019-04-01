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
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
)

//DeRegisterNodeFromMaster to deregister the node from master
func DeRegisterNodeFromMaster(ctx *TestContext, nodehandler, nodename string) error {
	err, resp := SendHttpRequest(http.MethodDelete, ctx.Cfg.ApiServer+nodehandler+"/"+nodename)
	if err != nil {
		Failf("Sending SenHttpRequest failed: %v", err)
		return err
	}
	defer resp.Body.Close()
	Expect(resp.StatusCode).Should(Equal(http.StatusOK))

	return nil
}

//GenerateNodeReqBody to generate the node request body
func GenerateNodeReqBody(nodeid, nodeselector string) (error, map[string]interface{}) {
	var temp map[string]interface{}

	body := fmt.Sprintf(`{"kind": "Node","apiVersion": "v1","metadata": {"name": "%s","labels": {"name": "edgenode", "disktype":"%s"}}}`, nodeid, nodeselector)
	err := json.Unmarshal([]byte(body), &temp)
	if err != nil {
		Failf("Unmarshal body failed: %v", err)
		return err, temp
	}

	return nil, temp
}

//RegisterNodeToMaster to register node to master
func RegisterNodeToMaster(ctx *TestContext, UID, nodehandler, nodeselector string) error {
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
	req, err := http.NewRequest(http.MethodPost, ctx.Cfg.ApiServer+nodehandler, BodyBuf)
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

//CheckNodeReadyStatus to get node status
func CheckNodeReadyStatus(ctx *TestContext, nodehandler, nodename string) string {
	var node v1.Node
	var nodeStatus = "unknown"
	err, resp := SendHttpRequest(http.MethodGet, ctx.Cfg.ApiServer+nodehandler+"/"+nodename)
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

//CheckNodeDeleteStatus to check node delete status
func CheckNodeDeleteStatus(ctx *TestContext, nodehandler, nodename string) int {
	err, resp := SendHttpRequest(http.MethodGet, ctx.Cfg.ApiServer+nodehandler+"/"+nodename)
	if err != nil {
		Failf("Sending SenHttpRequest failed: %v", err)
		return -1
	}
	defer resp.Body.Close()
	return resp.StatusCode
}
