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
)

const (
	UbuntuOSType = "ubuntu"
	CentOSType   = "centos"

	DefaultDownloadURL = "https://download.docker.com"
	DockerPreqReqList  = "apt-transport-https ca-certificates curl gnupg-agent software-properties-common"

	KubernetesDownloadURL = "https://apt.kubernetes.io/"
	KubernetesGPGURL      = "https://packages.cloud.google.com/apt/doc/apt-key.gpg"

	KubeEdgeDownloadURL     = "https://github.com/kubeedge/kubeedge/releases/download"
	KubeEdgeConfigPath      = "/etc/kubeedge/"
	KubeEdgeBinaryName      = "edge_core"
	KubeEdgeDefaultCertPath = KubeEdgeConfigPath + "certs/"
	KubeEdgeConfigEdgeYaml  = KubeEdgeConfigPath + "kubeedge/edge/conf/edge.yaml"
	KubeEdgeToReplaceKey1   = "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e"
	KubeEdgeToReplaceKey2   = "0.0.0.0"
	KubeEdgeConfigNodeJSON  = KubeEdgeConfigPath + "kubeedge/edge/conf/node.json"
	KubeEdgeNodeJSONContent = `{
		"kind": "Node",
		"apiVersion": "v1",
		"metadata": {
		  "name": "fb4ebb70-2783-42b8-b3ef-63e2fd6d242e",
		  "labels": {
			"name": "edge-node"
		  }
		}
	  }`
)

type InstallState uint8

const (
	NewInstallRequired InstallState = iota
	AlreadySameVersionExist
	DefVerInstallRequired
	VersionNAInRepo
)

type ToolsInstaller interface {
	InstallTools() error
	TearDown() error
}

type OSTypeInstaller interface {
	IsToolVerInRepo(string, string) (bool, error)
	IsDockerInstalled(string) (InstallState, error)
	InstallDocker() error
	InstallMQTT() error
	IsK8SComponentInstalled(string, string) (InstallState, error)
	InstallK8S() error
	InstallKubeEdge() error
	SetDockerVersion(string)
	SetK8SVersionAndIsNodeFlag(version string, flag bool)
	SetKubeEdgeVersion(string)
	RunEdgeCore() error
	KillEdgeCore() error
}

type Common struct {
	OSTypeInstaller
	OSVersion   string
	ToolVersion string
}

func (co *Common) SetOSInterface(intf OSTypeInstaller) {
	co.OSTypeInstaller = intf
}

type Command struct {
	Cmd    *exec.Cmd
	StdOut []byte
	StdErr []byte
}

func (cm *Command) ExecuteCommand() {
	var err error
	cm.StdOut, err = cm.Cmd.Output()
	if err != nil {
		fmt.Println("Output failed: ", err)
	}
}

func (cm Command) GetStdOutput() string {
	if len(cm.StdOut) != 0 {
		return strings.TrimRight(string(cm.StdOut), "\n")
	}
	return ""
}

func (cm Command) GetStdErr() string {
	if len(cm.StdErr) != 0 {
		return strings.TrimRight(string(cm.StdErr), "\n")
	}
	return ""
}

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

func GetOSVersion() string {
	c := &Command{Cmd: exec.Command("sh", "-c", ". /etc/os-release && echo $ID")}
	c.ExecuteCommand()
	return c.GetStdOutput()
}

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
