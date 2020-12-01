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

package util

import (
	"fmt"

	"github.com/blang/semver"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

//CentOS struct objects shall have information of the tools version to be installed
//on Hosts having CentOS OS.
//It implements OSTypeInstaller interface
type CentOS struct {
	KubeEdgeVersion semver.Version
	IsEdgeNode      bool //True - Edgenode False - Cloudnode
}

//SetKubeEdgeVersion sets the KubeEdge version for the objects instance
func (c *CentOS) SetKubeEdgeVersion(version semver.Version) {
	c.KubeEdgeVersion = version
}

//InstallMQTT checks if MQTT is already installed and running, if not then install it from OS repo
//Information is used from https://www.digitalocean.com/community/tutorials/how-to-install-and-secure-the-mosquitto-mqtt-messaging-broker-on-centos-7
func (c *CentOS) InstallMQTT() error {
	// check MQTT
	cmd := NewCommand("ps aux |awk '/mosquitto/ {print $1}' | awk '/mosquit/ {print}'")
	if err := cmd.Exec(); err != nil {
		return err
	}

	if stdout := cmd.GetStdOut(); stdout != "" {
		fmt.Println("Host has", stdout, "already installed and running. Hence skipping the installation steps !!!")
		return nil
	}

	// install MQTT
	for _, command := range []string{
		"yum -y install epel-release",
		"yum -y install mosquitto",
		"systemctl start mosquitto",
		"systemctl enable mosquitto",
	} {
		cmd := NewCommand(command)
		if err := cmd.Exec(); err != nil {
			return err
		}
	}
	fmt.Println("install MQTT service successfully.")

	return nil
}

//IsK8SComponentInstalled checks if said K8S version is already installed in the host
func (c *CentOS) IsK8SComponentInstalled(kubeConfig, master string) error {
	return isK8SComponentInstalled(kubeConfig, master)
}

//InstallKubeEdge downloads the provided version of KubeEdge.
//Untar's in the specified location /etc/kubeedge/ and then copies
//the binary to excecutables' path (eg: /usr/local/bin)
func (c *CentOS) InstallKubeEdge(options types.InstallOptions) error {
	arch := "amd64"
	cmd := NewCommand("arch")
	if err := cmd.Exec(); err != nil {
		return err
	}
	result := cmd.GetStdOut()
	switch result {
	case "armv7l":
		arch = "arm"
	case "aarch64":
		arch = "arm64"
	case "x86_64":
		arch = "amd64"
	default:
		return fmt.Errorf("can't support this architecture of CentOS: %s", result)
	}

	return installKubeEdge(options, arch, c.KubeEdgeVersion)
}

//RunEdgeCore sets the environment variable GOARCHAIUS_CONFIG_PATH for the configuration path
//and the starts edgecore with logs being captured
func (c *CentOS) RunEdgeCore() error {
	return runEdgeCore(c.KubeEdgeVersion)
}

//KillKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func (c *CentOS) KillKubeEdgeBinary(proc string) error {
	return killKubeEdgeBinary(proc)
}

//IsKubeEdgeProcessRunning checks if the given process is running or not
func (c *CentOS) IsKubeEdgeProcessRunning(proc string) (bool, error) {
	return isKubeEdgeProcessRunning(proc)
}

//IsKubeEdgeProcessRunning checks if the given process is running or not
func (c *CentOS) IsProcessRunning(proc string) (bool, error) {
	return isKubeEdgeProcessRunning(proc)
}
