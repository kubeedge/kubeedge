package util

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

type KubeEdgeInstTool struct {
	Common
	//CertPath       string
	K8SApiServerIP string
}

func (ku *KubeEdgeInstTool) InstallTools() error {
	ku.SetOSInterface(GetOSInterface())
	ku.SetKubeEdgeVersion(ku.ToolVersion)

	err := ku.InstallKubeEdge()
	if err != nil {
		return err
	}

	err = ku.modifyEdgeYamlNodeJSON()
	if err != nil {
		return err
	}

	err = ku.RunEdgeCore()
	if err != nil {
		return err
	}
	return nil
}

// func (ku *KubeEdgeInstTool) setCertPath(path string) {
// 	if path != "" {
// 		ku.CertPath = path
// 	} else {
// 		ku.CertPath = KubeEdgeDefaultCertPath
// 	}
// }

func (ku *KubeEdgeInstTool) SetK8SApiServerIP(server string) error {
	if server == "" {
		return fmt.Errorf("K8S API Server IP not provided")
	}
	ku.K8SApiServerIP = server
	//TODO: IP format validation should be done
	return nil
}

func (ku *KubeEdgeInstTool) modifyEdgeYamlNodeJSON() error {

	//Update edge.yaml, server ip for websocket communication

	edgeYaml, err := ioutil.ReadFile(KubeEdgeConfigEdgeYaml)
	if err != nil {
		return err
	}

	serverIPAddr := strings.Split(ku.K8SApiServerIP, ":")[0]
	rep1 := bytes.Replace(edgeYaml, []byte(KubeEdgeToReplaceKey2), []byte(serverIPAddr), -1)

	//Update edge.yaml, node ip to act as node id

	edgeIP, err := GetInterfaceIP()
	if err != nil {
		return err
	}
	rep2 := bytes.Replace(rep1, []byte(KubeEdgeToReplaceKey1), []byte(edgeIP), -1)

	if err = ioutil.WriteFile(KubeEdgeConfigEdgeYaml, rep2, 0666); err != nil {
		return err
	}

	nodeJSON, err := ioutil.ReadFile(KubeEdgeConfigNodeJSON)
	if err != nil {
		return err
	}

	//Update node.json, node ip to act as node id
	rep := bytes.Replace(nodeJSON, []byte(KubeEdgeToReplaceKey1), []byte(edgeIP), -1)

	if err = ioutil.WriteFile(KubeEdgeConfigNodeJSON, rep, 0666); err != nil {
		return err
	}

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
