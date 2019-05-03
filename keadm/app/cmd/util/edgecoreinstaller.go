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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

//KubeEdgeInstTool embedes Common struct and contains cloud node ip:port information
//It implements ToolsInstaller interface
type KubeEdgeInstTool struct {
	Common
	//CertPath       string
	K8SApiServerIP string
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

	//Create edge.yaml
	//Update edge.yaml, node ip to act as node id
	//Get node ip address from the interface
	edgeIP, err := GetInterfaceIP()
	if err != nil {
		return err
	}
	change1 := bytes.Replace(EdgeYaml, []byte(KubeEdgeToReplaceKey1), []byte(edgeIP), -1)

	serverIPAddr := strings.Split(ku.K8SApiServerIP, ":")[0]
	change2 := bytes.Replace(change1, []byte(KubeEdgeToReplaceKey2), []byte(serverIPAddr), -1)

	change3 := bytes.Replace(change2, []byte("2.0.0"), []byte(ku.ToolVersion), -1)

	if err = ioutil.WriteFile(KubeEdgeConfigEdgeYaml, change3, 0666); err != nil {
		return err
	}

	//Create logging.yaml
	if err = ioutil.WriteFile(KubeEdgeConfigLoggingYaml, EdgeLoggingYaml, 0666); err != nil {
		return err
	}
	//Create modules.yaml
	if err = ioutil.WriteFile(KubeEdgeConfigModulesYaml, EdgeModulesYaml, 0666); err != nil {
		return err
	}

	//Create node.json
	rep := bytes.Replace([]byte(EdgeNodeJSONContent), []byte(KubeEdgeToReplaceKey1), []byte(edgeIP), -1)

	if err = ioutil.WriteFile(KubeEdgeConfigNodeJSON, rep, 0666); err != nil {
		return err
	}

	//Add edge node in api-server using kubectl command
	//kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/node.json -s http://192.168.20.50:8080
	nodeJSONApply := fmt.Sprintf("kubectl apply -f %s -s http://%s", KubeEdgeConfigNodeJSON, ku.K8SApiServerIP)
	cmd := &Command{Cmd: exec.Command("sh", "-c", nodeJSONApply)}
	err = cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	return nil
}

//TearDown method will remove the edge node from api-server and stop edge_core process
func (ku *KubeEdgeInstTool) TearDown() error {

	ku.SetOSInterface(GetOSInterface())

	//Remove the edge from api server using kubectl command, like below
	//kubectl delete -f $GOPATH/src/github.com/kubeedge/kubeedge/build/node.json -s http://192.168.20.50:8080
	nodeJSONDelete := fmt.Sprintf("kubectl delete -f %s -s http://%s", KubeEdgeConfigNodeJSON, ku.K8SApiServerIP)
	cmd := &Command{Cmd: exec.Command("sh", "-c", nodeJSONDelete)}
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	//Kill edge core process
	ku.KillKubeEdgeBinary(KubeEdgeBinaryName)

	return nil
}
