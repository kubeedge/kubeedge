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
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/spf13/pflag"
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
	KubeEdgeToReplaceKey1     = "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e"
	KubeEdgeToReplaceKey2     = "0.0.0.0"
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

	KubeAPIServerName = "kube-apiserver"
)

//InstallState enum set used for verifying a tool version is installed in host
type InstallState uint8

//Difference enum values for type InstallState
const (
	NewInstallRequired InstallState = iota
	AlreadySameVersionExist
	DefVerInstallRequired
	VersionNAInRepo
)

//ModuleRunning is defined to know the running status of KubeEdge components
type ModuleRunning uint8

//Different possible values for ModuleRunning type
const (
	NoneRunning ModuleRunning = iota
	KubeEdgeCloudRunning
	KubeEdgeEdgeRunning
)

//ToolsInstaller interface for tools with install and teardown methods.
type ToolsInstaller interface {
	InstallTools() error
	TearDown() error
}

//OSTypeInstaller interface for methods to be executed over a specified OS distribution type
type OSTypeInstaller interface {
	IsToolVerInRepo(string, string) (bool, error)
	IsDockerInstalled(string) (InstallState, error)
	InstallDocker() error
	InstallMQTT() error
	IsK8SComponentInstalled(string, string) (InstallState, error)
	InstallK8S() error
	StartK8Scluster() error
	InstallKubeEdge() error
	SetDockerVersion(string)
	SetK8SVersionAndIsNodeFlag(version string, flag bool)
	SetKubeEdgeVersion(string)
	RunEdgeCore() error
	KillKubeEdgeBinary(string) error
	IsKubeEdgeProcessRunning(string) (bool, error)
}

//FlagData stores value and default value of the flags used in this command
type FlagData struct {
	Val    interface{}
	DefVal interface{}
}

//AddToolVals gets the value and default values of each flags and collects them in temporary cache
func AddToolVals(f *pflag.Flag, flagData map[string]FlagData) {
	flagData[f.Name] = FlagData{Val: f.Value.String(), DefVal: f.DefValue}
}

//CheckIfAvailable checks is val of a flag is empty then return the default value
func CheckIfAvailable(val, defval string) string {
	if val == "" {
		return defval
	}
	return val
}

//NodeMetaDataLabels defines
type NodeMetaDataLabels struct {
	Name string
}

//NodeMetaDataSt defines
type NodeMetaDataSt struct {
	Name   string
	Labels NodeMetaDataLabels
}

//NodeDefinition defines
type NodeDefinition struct {
	Kind       string
	APIVersion string
	MetaData   NodeMetaDataSt
}

//Common struct contains OS and Tool version properties and also embeds OS interface
type Common struct {
	OSTypeInstaller
	OSVersion   string
	ToolVersion string
}

//SetOSInterface defines a method to set the implemtation of the OS interface
func (co *Common) SetOSInterface(intf OSTypeInstaller) {
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

//GetInterfaceIP gets the interface ip address, this command helps in getting the edge node ip
func GetInterfaceIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("Not able to get interfaces")
}

//GetOSInterface helps in returning OS specific object which implements OSTypeInstaller interface.
func GetOSInterface() OSTypeInstaller {

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
func IsKubeEdgeController() (ModuleRunning, error) {
	osType := GetOSInterface()
	edgeControllerRunning, err := osType.IsKubeEdgeProcessRunning(KubeCloudBinaryName)
	if err != nil {
		return NoneRunning, err
	}
	apiServerRunning, err := osType.IsKubeEdgeProcessRunning(KubeAPIServerName)
	if err != nil {
		return NoneRunning, err
	}
	//If any of edgecontroller or K8S API server is running, then we believe the node is cloud node
	if edgeControllerRunning || apiServerRunning {
		return KubeEdgeCloudRunning, nil
	}

	edgeCoreRunning, err := osType.IsKubeEdgeProcessRunning(KubeEdgeBinaryName)
	if err != nil {
		return NoneRunning, err
	}

	if false != edgeCoreRunning {
		return KubeEdgeEdgeRunning, nil
	}

	return NoneRunning, nil
}
