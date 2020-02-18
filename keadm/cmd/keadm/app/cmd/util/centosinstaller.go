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
	"fmt"
	"os/exec"
)

//CentOS struct objects shall have information of the tools version to be installed
//on Hosts having Ubuntu OS.
//It implements OSTypeInstaller interface
type CentOS struct {
	KubernetesVersion string
	KubeEdgeVersion   string
	IsEdgeNode        bool //True - Edgenode False - Cloudnode
}

//SetKubeEdgeVersion sets the KubeEdge version for the objects instance
func (c *CentOS) SetKubeEdgeVersion(version string) {
	c.KubeEdgeVersion = version
}

//InstallMQTT checks if MQTT is already installed and running, if not then install it from OS repo
//Information is used from https://www.digitalocean.com/community/tutorials/how-to-install-and-secure-the-mosquitto-mqtt-messaging-broker-on-centos-7
func (c *CentOS) InstallMQTT() error {
	//yum -y install epel-release
	cmd := &Command{Cmd: exec.Command("sh", "-c", "yum -y install epel-release")}
	err := cmd.ExecuteCmdShowOutput()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	//yum -y install mosquitto
	cmd = &Command{Cmd: exec.Command("sh", "-c", "yum -y install mosquitto")}
	err = cmd.ExecuteCmdShowOutput()
	stdout = cmd.GetStdOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	//systemctl start mosquitto
	cmd = &Command{Cmd: exec.Command("sh", "-c", "systemctl start mosquitto")}
	cmd.ExecuteCommand()
	stdout = cmd.GetStdOutput()
	errout = cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	//systemctl enable mosquitto
	cmd = &Command{Cmd: exec.Command("sh", "-c", "systemctl enable mosquitto")}
	cmd.ExecuteCommand()
	stdout = cmd.GetStdOutput()
	errout = cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	return nil
}

//IsK8SComponentInstalled checks if said K8S version is already installed in the host
func (c *CentOS) IsK8SComponentInstalled(component, defVersion string) error {
	// 	[root@localhost ~]# yum list installed | grep kubeadm | awk '{print $2}' | cut -d'-' -f 1
	// 1.14.1
	// [root@localhost ~]#
	// [root@localhost ~]# yum list installed | grep kubeadm
	// kubeadm.x86_64                          1.14.1-0                       @kubernetes
	// [root@localhost ~]#

	return nil
}

//InstallKubeEdge downloads the provided version of KubeEdge.
//Untar's in the specified location /etc/kubeedge/ and then copies
//the binary to excecutables' path (eg: /usr/local/bin)
func (c *CentOS) InstallKubeEdge() error {
	fmt.Println("InstallKubeEdge called")
	return nil
}

//RunEdgeCore sets the environment variable GOARCHAIUS_CONFIG_PATH for the configuration path
//and the starts edgecore with logs being captured
func (c *CentOS) RunEdgeCore() error {
	return nil
}

//KillKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func (c *CentOS) KillKubeEdgeBinary(proc string) error {
	return nil
}

//IsKubeEdgeProcessRunning checks if the given process is running or not
func (c *CentOS) IsKubeEdgeProcessRunning(proc string) (bool, error) {
	return false, nil
}
