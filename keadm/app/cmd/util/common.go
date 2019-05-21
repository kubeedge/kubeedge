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
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/spf13/pflag"

	types "github.com/kubeedge/kubeedge/keadm/app/cmd/common"
)

//Constants used by installers
const (
	UbuntuOSType = "ubuntu"
	CentOSType   = "centos"

	DefaultDownloadURL = "https://download.docker.com"
	DockerPreqReqList  = "apt-transport-https ca-certificates curl gnupg-agent software-properties-common"

	KubernetesDownloadURL = "https://apt.kubernetes.io/"
	KubernetesGPGURL      = "https://packages.cloud.google.com/apt/doc/apt-key.gpg"

	KubeEdgeDownloadURL       = "https://github.com/kubeedge/kubeedge/releases/download"
	KubeEdgePath              = "/etc/kubeedge/"
	KubeEdgeConfPath          = KubeEdgePath + "kubeedge/edge/conf"
	KubeEdgeBinaryName        = "edge_core"
	KubeEdgeDefaultCertPath   = KubeEdgePath + "certs/"
	KubeEdgeConfigEdgeYaml    = KubeEdgeConfPath + "/edge.yaml"
	KubeEdgeConfigNodeJSON    = KubeEdgeConfPath + "/node.json"
	KubeEdgeConfigLoggingYaml = KubeEdgeConfPath + "/logging.yaml"
	KubeEdgeConfigModulesYaml = KubeEdgeConfPath + "/modules.yaml"

	KubeEdgeCloudCertGenPath      = KubeEdgePath + "certgen.sh"
	KubeEdgeEdgeCertsTarFileName  = "certs.tgz"
	KubeEdgeEdgeCertsTarFilePath  = KubeEdgePath + "certs.tgz"
	KubeEdgeCloudConfPath         = KubeEdgePath + "kubeedge/cloud/conf"
	KubeEdgeControllerYaml        = KubeEdgeCloudConfPath + "/controller.yaml"
	KubeEdgeControllerLoggingYaml = KubeEdgeCloudConfPath + "/logging.yaml"
	KubeEdgeControllerModulesYaml = KubeEdgeCloudConfPath + "/modules.yaml"
	KubeCloudBinaryName           = "edgecontroller"
	KubeCloudApiserverYamlPath    = "/etc/kubernetes/manifests/kube-apiserver.yaml"
	KubeCloudReplaceIndex         = 25
	KubeCloudReplaceString        = "    - --insecure-bind-address=0.0.0.0\n"

	KubeAPIServerName          = "kube-apiserver"
	KubeEdgeHTTPProto          = "http"
	KubeEdgeHTTPSProto         = "https"
	KubeEdgeHTTPPort           = "8080"
	KubeEdgeHTTPSPort          = "6443"
	KubeEdgeHTTPRequestTimeout = 30
)

//AddToolVals gets the value and default values of each flags and collects them in temporary cache
func AddToolVals(f *pflag.Flag, flagData map[string]types.FlagData) {
	flagData[f.Name] = types.FlagData{Val: f.Value.String(), DefVal: f.DefValue}
}

//CheckIfAvailable checks is val of a flag is empty then return the default value
func CheckIfAvailable(val, defval string) string {
	if val == "" {
		return defval
	}
	return val
}

//Common struct contains OS and Tool version properties and also embeds OS interface
type Common struct {
	types.OSTypeInstaller
	OSVersion   string
	ToolVersion string
	KubeConfig  string
}

//SetOSInterface defines a method to set the implemtation of the OS interface
func (co *Common) SetOSInterface(intf types.OSTypeInstaller) {
	co.OSTypeInstaller = intf
}

//Command defines commands to be executed and captures std out and std error
type Command struct {
	Cmd    *exec.Cmd
	StdOut []byte
	StdErr []byte
}

//ExecuteCommand executes the command and captures the output in stdOut
func (cm *Command) ExecuteCommand() {
	var err error
	cm.StdOut, err = cm.Cmd.Output()
	if err != nil {
		fmt.Println("Output failed: ", err)
		cm.StdErr = []byte(err.Error())
	}
}

//GetStdOutput gets StdOut field
func (cm Command) GetStdOutput() string {
	if len(cm.StdOut) != 0 {
		return strings.TrimRight(string(cm.StdOut), "\n")
	}
	return ""
}

//GetStdErr gets StdErr field
func (cm Command) GetStdErr() string {
	if len(cm.StdErr) != 0 {
		return strings.TrimRight(string(cm.StdErr), "\n")
	}
	return ""
}

//ExecuteCmdShowOutput captures both StdOut and StdErr after exec.cmd().
//It helps in the commands where it takes some time for execution.
func (cm Command) ExecuteCmdShowOutput() error {
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutIn, _ := cm.Cmd.StdoutPipe()
	stderrIn, _ := cm.Cmd.StderrPipe()

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err := cm.Cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start because of error : %s", err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
		wg.Done()
	}()

	_, errStderr = io.Copy(stderr, stderrIn)
	wg.Wait()

	err = cm.Cmd.Wait()
	if err != nil {
		return fmt.Errorf("failed to run because of error : %s", err.Error())
	}
	if errStdout != nil || errStderr != nil {
		return fmt.Errorf("failed to capture stdout or stderr")
	}

	cm.StdOut, cm.StdErr = stdoutBuf.Bytes(), stderrBuf.Bytes()
	return nil
}

//GetOSVersion gets the OS name
func GetOSVersion() string {
	c := &Command{Cmd: exec.Command("sh", "-c", ". /etc/os-release && echo $ID")}
	c.ExecuteCommand()
	return c.GetStdOutput()
}

//GetOSInterface helps in returning OS specific object which implements OSTypeInstaller interface.
func GetOSInterface() types.OSTypeInstaller {

	switch GetOSVersion() {
	case UbuntuOSType:
		return &UbuntuOS{}
	case CentOSType:
		return &CentOS{}
	default:
	}
	return nil
}

//IsKubeEdgeController identifies if the node is having edge controller and k8s api-server already running.
//If so, then return true, else it can used as edge node and initialise it.
func IsKubeEdgeController() (types.ModuleRunning, error) {
	osType := GetOSInterface()
	edgeControllerRunning, err := osType.IsKubeEdgeProcessRunning(KubeCloudBinaryName)
	if err != nil {
		return types.NoneRunning, err
	}
	apiServerRunning, err := osType.IsKubeEdgeProcessRunning(KubeAPIServerName)
	if err != nil {
		return types.NoneRunning, err
	}
	//If any of edgecontroller or K8S API server is running, then we believe the node is cloud node
	if edgeControllerRunning || apiServerRunning {
		return types.KubeEdgeCloudRunning, nil
	}

	edgeCoreRunning, err := osType.IsKubeEdgeProcessRunning(KubeEdgeBinaryName)
	if err != nil {
		return types.NoneRunning, err
	}

	if false != edgeCoreRunning {
		return types.KubeEdgeEdgeRunning, nil
	}

	return types.NoneRunning, nil
}
