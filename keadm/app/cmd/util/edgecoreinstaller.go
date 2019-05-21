/*
Copyright 2019 The Kubeedge Authors.

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

package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	types "github.com/kubeedge/kubeedge/keadm/app/cmd/common"
)

//KubeEdgeInstTool embedes Common struct and contains cloud node ip:port information
//It implements ToolsInstaller interface
type KubeEdgeInstTool struct {
	Common
	//CertPath       string
	EdgeContrlrIP  string
	K8SApiServerIP string
	EdgeNodeID     string
	RuntimeType    string
}

//InstallTools downloads KubeEdge for the specified verssion
//and makes the required configuration changes and initiates edge_core.
func (ku *KubeEdgeInstTool) InstallTools() error {
	ku.SetOSInterface(GetOSInterface())
	ku.SetKubeEdgeVersion(ku.ToolVersion)

	err := ku.InstallKubeEdge()
	if err != nil {
		return err
	}

	err = ku.createEdgeConfigFiles()
	if err != nil {
		return err
	}

	err = ku.RunEdgeCore()
	if err != nil {
		return err
	}
	return nil
}

func (ku *KubeEdgeInstTool) createEdgeConfigFiles() error {

	//This makes sure the path is created, if it already exists also it is fine
	err := os.MkdirAll(KubeEdgeConfPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeConfPath)
	}

	// //Create edge.yaml
	//Update edge.yaml with a unique id against node id
	//If the user doesn't provide any edge ID on the command line, then it generates unique id and assigns it.
	edgeID := uuid.New().String()
	if "" != ku.EdgeNodeID {
		edgeID = ku.EdgeNodeID
	}

	serverIPAddr := "0.0.0.0"
	if "" != ku.EdgeContrlrIP {
		serverIPAddr = ku.EdgeContrlrIP
	}

	url := fmt.Sprintf("wss://%s:10000/%s/%s/events", serverIPAddr, types.DefaultProjectID, edgeID)
	edgeYaml := &types.EdgeYamlSt{EdgeHub: types.EdgeHubSt{WebSocket: types.WebSocketSt{URL: url}},
		EdgeD: types.EdgeDSt{Version: types.VendorK8sPrefix + ku.ToolVersion, RuntimeType: ku.RuntimeType}}

	if err = types.WriteEdgeYamlFile(KubeEdgeConfigEdgeYaml, edgeYaml); err != nil {
		return err
	}

	//Create logging.yaml
	if err = types.WriteEdgeLoggingYamlFile(KubeEdgeConfigLoggingYaml); err != nil {
		return err
	}
	//Create modules.yaml
	if err = types.WriteEdgeModulesYamlFile(KubeEdgeConfigModulesYaml); err != nil {
		return err
	}

	if "" != ku.K8SApiServerIP {
		if err := ku.addNodeToK8SAPIServer(edgeID, ku.K8SApiServerIP); err != nil {
			return err
		}
	} else {
		data := &v1.Node{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Node"},
			ObjectMeta: metav1.ObjectMeta{
				Name:   edgeID,
				Labels: map[string]string{"name": "edge-node", "node-role.kubernetes.io/edge": ""},
			}}

		respBytes, err := json.Marshal(data)
		if err != nil {
			return err
		}

		if err = ioutil.WriteFile(KubeEdgeConfigNodeJSON, respBytes, 0666); err != nil {
			return err
		}

		fmt.Println("KubeEdge Edge Node:", edgeID, "will be started")
	}

	return nil
}

func (ku *KubeEdgeInstTool) addNodeToK8SAPIServer(edgeid, server string) error {
	client := &http.Client{}

	data := &v1.Node{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Node"},
		ObjectMeta: metav1.ObjectMeta{
			Name:   edgeid,
			Labels: map[string]string{"name": "edge-node", "node-role.kubernetes.io/edge": ""},
		}}

	respBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	proto := KubeEdgeHTTPProto
	if KubeEdgeHTTPSPort == strings.Split(server, ":")[1] {
		proto = KubeEdgeHTTPSProto
	}

	kubeAPIServerURL := fmt.Sprintf("%s://%s:%s/api/v1/nodes", proto, strings.Split(server, ":")[0], strings.Split(server, ":")[1])
	req, err := http.NewRequest(http.MethodPost, kubeAPIServerURL, bytes.NewBuffer(respBytes))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Println("KubeEdge Edge Node:", edgeid, "successfully add to kube-apiserver, with operation status:", resp.Status)
		if err = ioutil.WriteFile(KubeEdgeConfigNodeJSON, respBytes, 0666); err != nil {
			return err
		}

	case http.StatusConflict:
		fmt.Println("KubeEdge Edge Node:", edgeid, "already exists and no change required")
	default:
		fmt.Println("KubeEdge Edge Node:", edgeid, "failed due to reasons mentioned in operation response", string(contents))
	}

	fmt.Println("Content", string(contents))
	return nil
}

func (ku *KubeEdgeInstTool) deleteNodeFromK8SAPIServer(server string) error {
	var byte io.Reader
	client := &http.Client{}

	proto := KubeEdgeHTTPProto
	if KubeEdgeHTTPSPort == strings.Split(server, ":")[1] {
		proto = KubeEdgeHTTPSProto
	}

	fileData, err := ioutil.ReadFile(KubeEdgeConfigNodeJSON)
	if err != nil {
		return err
	}

	node := &v1.Node{}
	err = json.Unmarshal(fileData, &node)
	if err != nil {
		return err
	}

	kubeAPIServerURL := fmt.Sprintf("%s://%s:%s/api/v1/nodes/%s", proto, strings.Split(server, ":")[0], strings.Split(server, ":")[1], node.GetName())
	req, err := http.NewRequest(http.MethodDelete, kubeAPIServerURL, byte)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Println(resp.Status, ": Node", node.GetName(), "successfully deleted from kube-apiserver")
	case http.StatusNotFound:
		fmt.Println(resp.Status, "Node already already deleted or never existed")
	default:
		fmt.Println(resp.Status, "Content", string(contents))
	}

	fmt.Println("Content", string(contents))
	return nil
}

//TearDown method will remove the edge node from api-server and stop edge_core process
func (ku *KubeEdgeInstTool) TearDown() error {

	ku.SetOSInterface(GetOSInterface())

	if "" != ku.K8SApiServerIP {
		//Remove the edge from api server using kubectl command, like below
		if err := ku.deleteNodeFromK8SAPIServer(ku.K8SApiServerIP); err != nil {
			return err
		}
	}

	//Kill edge core process
	ku.KillKubeEdgeBinary(KubeEdgeBinaryName)

	return nil
}
