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

package common

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
	"github.com/kubeedge/kubeedge/tests/stubs/common/types"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

//K8s resource handlers
const (
	AppHandler        = "/api/v1/namespaces/default/pods"
	NodeHandler       = "/api/v1/nodes"
	DeploymentHandler = "/apis/apps/v1/namespaces/default/deployments"
	ConfigmapHandler  = "/api/v1/namespaces/default/configmaps"
	ServiceHandler    = "/api/v1/namespaces/default/services"
	NodelabelKey      = "k8snode"
	NodelabelVal      = "kb-perf-node"
)

var (
	chconfigmapRet      = make(chan error)
	Deployments         []string
	NodeInfo            = make(map[string][]string)
	CloudConfigMap      string
	CloudCoreDeployment string
	ToTaint             bool
	IsQuicProtocol      bool
)

func HandleCloudDeployment(cloudConfigMap, cloudCoreDeployment, apiserver2, confighdl, deploymenthdl, imgURL string, nodelimit int) error {
	nodes := strconv.FormatInt(int64(nodelimit), 10)
	cmd := exec.Command("bash", "-x", "scripts/update_configmap.sh", "create_cloud_config", "", "http://192.168.20.128:8080", cloudConfigMap, nodes)
	err := utils.PrintCombinedOutput(cmd)
	Expect(err).Should(BeNil())
	go utils.HandleConfigmap(chconfigmapRet, http.MethodPost, confighdl, false)
	ret := <-chconfigmapRet
	Expect(ret).To(BeNil())
	utils.ProtocolQuic = IsQuicProtocol
	//Handle cloudCore deployment
	go utils.HandleDeployment(true, false, http.MethodPost, deploymenthdl, cloudCoreDeployment, imgURL, "", cloudConfigMap, 1)

	return nil
}

func CreateConfigMapforEdgeCore(cloudhub, cmHandler, nodeHandler string, numOfNodes int){
	//Create edgecore configMaps based on the users choice of edgecore deployment.
	for i := 0; i < numOfNodes; i++ {
		nodeName := "perf-node-" + utils.GetRandomString(10)
		nodeSelector := "node-" + utils.GetRandomString(5)
		configmap := "edgecore-configmap-" + utils.GetRandomString(5)
		//Register EdgeNodes to K8s Master
		go utils.RegisterNodeToMaster(nodeName, nodeHandler, nodeSelector)
		cmd := exec.Command("bash", "-x", "scripts/update_configmap.sh", "create_edge_config", nodeName, cloudhub, configmap)
		err := utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())
		//Create ConfigMaps for Each EdgeNode created
		go utils.HandleConfigmap(chconfigmapRet, http.MethodPost, cmHandler, true)
		ret := <-chconfigmapRet
		Expect(ret).To(BeNil())
		//Store the ConfigMap against each edgenode
		NodeInfo[nodeName] = append(NodeInfo[nodeName], configmap, nodeSelector)
	}
}

func HandleEdgeCorePodDeployment(depHandler, imgURL, podHandler, nodeHandler string, numOfNodes int) v1.PodList  {
	replica := 1
	//Create edgeCore deployments as per users configuration
	for _, configmap := range NodeInfo {
		UID := "edgecore-deployment-" + utils.GetRandomString(5)
		go utils.HandleDeployment(false, true, http.MethodPost, depHandler, UID, imgURL, "", configmap[0], replica)
		Deployments = append(Deployments, UID)
	}
	time.Sleep(2 * time.Second)
	podlist, err := utils.GetPods(podHandler, "")
	Expect(err).To(BeNil())
	utils.CheckPodRunningState(podHandler, podlist)

	//Check All EdgeNode are in Running state
	Eventually(func() int {
		count := 0
		for edgenodeName, _ := range NodeInfo {
			status := utils.CheckNodeReadyStatus(nodeHandler, edgenodeName)
			utils.Info("Node Name: %v, Node Status: %v", edgenodeName, status)
			if status == "Running" {
				count++
			}
		}
		return count
	}, "1200s", "2s").Should(Equal(numOfNodes), "Nodes register to the k8s master is unsuccessfull !!")

	return podlist
}

func HandleEdgeDeployment(cloudhub, depHandler, nodeHandler, cmHandler, imgURL, podHandler string, numOfNodes int) v1.PodList {
	CreateConfigMapforEdgeCore(cloudhub, cmHandler, nodeHandler, numOfNodes)
	podlist := HandleEdgeCorePodDeployment(depHandler, imgURL, podHandler, nodeHandler, numOfNodes)
	return podlist
}

func DeleteEdgeDeployments(apiServerForRegisterNode, apiServerForDeployments string, nodes int) {
	//delete confogMap
	for _, configmap := range NodeInfo {
		go utils.HandleConfigmap(chconfigmapRet, http.MethodDelete, apiServerForDeployments+ConfigmapHandler+"/"+configmap[0], false)
		ret := <-chconfigmapRet
		Expect(ret).To(BeNil())

	}
	//delete edgenode deployment
	for _, depName := range Deployments {
		go utils.HandleDeployment(true, true, http.MethodDelete, apiServerForDeployments+DeploymentHandler+"/"+depName, "", "", "", "", 0)
	}
	//delete edgenodes
	for edgenodeName, _ := range NodeInfo {
		err := utils.DeRegisterNodeFromMaster(apiServerForRegisterNode+NodeHandler, edgenodeName)
		if err != nil {
			utils.Failf("DeRegisterNodeFromMaster failed: %v", err)
		}
	}
	//Verify deployments, configmaps, nodes are deleted successfully
	Eventually(func() int {
		count := 0
		for _, depName := range Deployments {
			statusCode := utils.VerifyDeleteDeployment(apiServerForDeployments + DeploymentHandler + "/" + depName)
			if statusCode == 404 {
				count++
			}
		}
		return count
	}, "60s", "4s").Should(Equal(len(Deployments)), "EdgeNode deployments delete unsuccessfull !!")

	Eventually(func() int {
		count := 0
		for _, configmap := range NodeInfo {
			statusCode, _ := utils.GetConfigmap(apiServerForDeployments + ConfigmapHandler + "/" + configmap[0])
			if statusCode == 404 {
				count++
			}
		}
		return count
	}, "60s", "4s").Should(Equal(len(Deployments)), "EdgeNode configMaps delete unsuccessfull !!")

	Eventually(func() int {
		count := 0
		for edgenodeName, _ := range NodeInfo {
			status := utils.CheckNodeDeleteStatus(apiServerForRegisterNode+NodeHandler, edgenodeName)
			utils.Info("Node Name: %v, Node Status: %v", edgenodeName, status)
			if status == 404 {
				count++
			}
		}
		return count
	}, "60s", "4s").Should(Equal(nodes), "EdgeNode deleton is unsuccessfull !!")
	//Cleanup globals
	NodeInfo = map[string][]string{}
	Deployments = nil
}

func DeleteCloudDeployment(apiserver string) {
	//delete cloud deployment
	go utils.HandleDeployment(true, true, http.MethodDelete, apiserver+DeploymentHandler+"/"+CloudCoreDeployment, "", "", "", "", 0)
	//delete cloud configMap
	go utils.HandleConfigmap(chconfigmapRet, http.MethodDelete, apiserver+ConfigmapHandler+"/"+CloudConfigMap, false)
	ret := <-chconfigmapRet
	Expect(ret).To(BeNil())
	//delete cloud svc
	StatusCode := utils.DeleteSvc(apiserver + ServiceHandler + "/" + CloudCoreDeployment)
	Expect(StatusCode).Should(Equal(http.StatusOK))
}

func ApplyLabel(nodeHandler string) error {
	var isMasterNode bool
	nodes := utils.GetNodes(nodeHandler)
	for _, node := range nodes.Items {
		isMasterNode = false
		for key, _ := range node.Labels {
			if strings.Contains(key, "node-role.kubernetes.io/master") {
				isMasterNode = true
				break
			}
		}
		if isMasterNode == false {
			utils.ApplyLabelToNode(nodeHandler+"/"+node.Name, NodelabelKey, NodelabelVal)
		}
	}
	return nil
}

// AddFakePod adds a fake pod
func AddFakePod(ControllerHubURL string, pod types.FakePod) {
	reqBody, err := json.Marshal(pod)
	if err != nil {
		utils.Failf("Unmarshal HTTP Response has failed: %v", err)
	}

	err, resp := SendHttpRequest(http.MethodPost,
		ControllerHubURL+constants.PodResource,
		bytes.NewBuffer(reqBody))
	if err != nil {
		utils.Failf("Frame HTTP request failed: %v", err)
	}

	if resp != nil {
		defer resp.Body.Close()

		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			utils.Failf("HTTP Response reading has failed: %v", err)
		}

		if contents != nil {
			utils.Info("AddPod response: %v", contents)
		} else {
			utils.Info("AddPod response: nil")
		}
	}
}

// DeleteFakePod deletes a fake pod
func DeleteFakePod(ControllerHubURL string, pod types.FakePod) {
	err, resp := SendHttpRequest(http.MethodDelete,
		ControllerHubURL+constants.PodResource+
			"?name="+pod.Name+"&namespace="+pod.Namespace+"&nodename="+pod.NodeName,
		nil)
	if err != nil {
		utils.Failf("Frame HTTP request failed: %v", err)
	}

	if resp != nil {
		defer resp.Body.Close()

		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			utils.Failf("HTTP Response reading has failed: %v", err)
		}

		if contents != nil {
			utils.Info("DeletePod response: %v", contents)
		} else {
			utils.Info("DeletePod response: nil")
		}
	}
}

// ListFakePods lists all fake pods
func ListFakePods(ControllerHubURL string) []types.FakePod {
	pods := []types.FakePod{}
	err, resp := SendHttpRequest(http.MethodGet, ControllerHubURL+constants.PodResource, nil)
	if err != nil {
		utils.Failf("Frame HTTP request failed: %v", err)
	}

	if resp != nil {
		defer resp.Body.Close()

		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			utils.Failf("HTTP Response reading has failed: %v", err)
		}

		err = json.Unmarshal(contents, &pods)
		if err != nil {
			utils.Failf("Unmarshal message content with error: %s", err)
		}
	}

	utils.Info("ListPods result: %d", len(pods))
	return pods
}

// SendHttpRequest launches a http request
func SendHttpRequest(method, reqApi string, body io.Reader) (error, *http.Response) {
	var resp *http.Response
	client := &http.Client{}
	req, err := http.NewRequest(method, reqApi, body)
	if err != nil {
		utils.Failf("Frame HTTP request failed: %v", err)
		return err, resp
	}
	req.Header.Set("Content-Type", "application/json")
	t := time.Now()
	resp, err = client.Do(req)
	if err != nil {
		utils.Failf("HTTP request is failed :%v", err)
		return err, resp
	}
	if resp != nil {
		utils.Info("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	}
	return nil, resp
}
