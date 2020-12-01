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

const downloadRetryTimes int = 3

//UbuntuOS struct objects shall have information of the tools version to be installed
//on Hosts having Ubuntu OS.
//It implements OSTypeInstaller interface
type UbuntuOS struct {
	KubeEdgeVersion semver.Version
	IsEdgeNode      bool //True - Edgenode False - Cloudnode
}

//SetKubeEdgeVersion sets the KubeEdge version for the objects instance
func (u *UbuntuOS) SetKubeEdgeVersion(version semver.Version) {
	u.KubeEdgeVersion = version
}

//InstallMQTT checks if MQTT is already installed and running, if not then install it from OS repo
func (u *UbuntuOS) InstallMQTT() error {
	cmd := NewCommand("ps aux |awk '/mosquitto/ {print $1}' | awk '/mosquit/ {print}'")
	if err := cmd.Exec(); err != nil {
		return err
	}

	if stdout := cmd.GetStdOut(); stdout != "" {
		fmt.Println("Host has", stdout, "already installed and running. Hence skipping the installation steps !!!")
		return nil
	}

	//Install mqttInst
	cmd = NewCommand("apt-get install -y --allow-change-held-packages --allow-downgrades mosquitto")
	if err := cmd.Exec(); err != nil {
		return err
	}
	fmt.Println(cmd.GetStdOut())

	fmt.Println("MQTT is installed in this host")

	return nil
}

// IsK8SComponentInstalled checks if said K8S version is already installed in the host
func (u *UbuntuOS) IsK8SComponentInstalled(kubeConfig, master string) error {
	return isK8SComponentInstalled(kubeConfig, master)
}

// InstallKubeEdge downloads the provided version of KubeEdge.
// Untar's in the specified location /etc/kubeedge/ and then copies
// the binary to excecutables' path (eg: /usr/local/bin)
func (u *UbuntuOS) InstallKubeEdge(options types.InstallOptions) error {
	arch, err := getSystemArch()
	if err != nil {
		return err
	}

	return installKubeEdge(options, arch, u.KubeEdgeVersion)
}

//RunEdgeCore sets the environment variable GOARCHAIUS_CONFIG_PATH for the configuration path
//and the starts edgecore with logs being captured
func (u *UbuntuOS) RunEdgeCore() error {
	return runEdgeCore(u.KubeEdgeVersion)
}

//KillKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func (u *UbuntuOS) KillKubeEdgeBinary(proc string) error {
	return killKubeEdgeBinary(proc)
}

//IsKubeEdgeProcessRunning checks if the given process is running or not
func (u *UbuntuOS) IsKubeEdgeProcessRunning(proc string) (bool, error) {
	return isKubeEdgeProcessRunning(proc)
}

func getSystemArch() (string, error) {
	cmd := NewCommand("dpkg --print-architecture")
	if err := cmd.Exec(); err != nil {
		return "", err
	}

	return cmd.GetStdOut(), nil
}

func (u *UbuntuOS) IsProcessRunning(proc string) (bool, error) {
	return isKubeEdgeProcessRunning(proc)
}
