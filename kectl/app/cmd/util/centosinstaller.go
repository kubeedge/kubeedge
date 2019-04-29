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
)

//CentOS struct objects shall have information of the tools version to be installed
//on Hosts having Ubuntu OS.
//It implements OSTypeInstaller interface
type CentOS struct {
	DockerVersion     string
	KubernetesVersion string
	KubeEdgeVersion   string
	IsEdgeNode        bool //True - Edgenode False - Cloudnode
}

//SetDockerVersion sets the Docker version for the objects instance
func (c *CentOS) SetDockerVersion(version string) {
	c.DockerVersion = version
}

//SetK8SVersionAndIsNodeFlag sets the K8S version for the objects instance
//It also sets if this host shall act as edge node or not
func (c *CentOS) SetK8SVersionAndIsNodeFlag(version string, flag bool) {
	c.KubernetesVersion = version
	c.IsEdgeNode = flag
}

//SetKubeEdgeVersion sets the KubeEdge version for the objects instance
func (c *CentOS) SetKubeEdgeVersion(version string) {
	c.KubeEdgeVersion = version
}

//IsDockerInstalled checks if docker is installed in the host or not
func (c *CentOS) IsDockerInstalled(string) (InstallState, error) {

	return VersionNAInRepo, nil
}

//InstallDocker will install the specified docker in the host
func (c *CentOS) InstallDocker() error {
	fmt.Println("InstallDocker called")
	return nil
}

//IsToolVerInRepo checks if the tool mentioned in available in OS repo or not
func (c *CentOS) IsToolVerInRepo(toolName, version string) (bool, error) {
	fmt.Println("IsToolVerInRepo called")
	return false, nil
}

//InstallMQTT checks if MQTT is already installed and running, if not then install it from OS repo
func (c *CentOS) InstallMQTT() error {
	fmt.Println("InstallMQTT called")
	return nil
}

//IsK8SComponentInstalled checks if said K8S version is already installed in the host
func (c *CentOS) IsK8SComponentInstalled(component, defVersion string) (InstallState, error) {
	return VersionNAInRepo, nil
}

//InstallK8S will install kubeadm, kudectl and kubelet for the cloud node
//and only kubectl for edge node
func (c *CentOS) InstallK8S() error {
	fmt.Println("InstallK8S called")
	return nil
}

//InstallKubeEdge downloads the provided version of KubeEdge.
//Untar's in the specified location /etc/kubeedge/ and then copies
//the binary to /usr/local/bin path.
func (c *CentOS) InstallKubeEdge() error {
	fmt.Println("InstallKubeEdge called")
	return nil
}

//RunEdgeCore sets the environment variable GOARCHAIUS_CONFIG_PATH for the configuration path
//and the starts edge_core with logs being captured
func (c *CentOS) RunEdgeCore() error {
	return nil
}

//KillEdgeCore will search for edge_core process and forcefully kill it
func (c *CentOS) KillEdgeCore() error {
	return nil
}
