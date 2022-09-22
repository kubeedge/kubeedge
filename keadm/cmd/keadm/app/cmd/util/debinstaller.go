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

// DebOS struct objects shall have information of the tools version to be installed
// on Hosts having Ubuntu OS.
// It implements OSTypeInstaller interface
type DebOS struct {
	KubeEdgeVersion semver.Version
	IsEdgeNode      bool //True - Edgenode False - Cloudnode
}

// SetKubeEdgeVersion sets the KubeEdge version for the objects instance
func (d *DebOS) SetKubeEdgeVersion(version semver.Version) {
	d.KubeEdgeVersion = version
}

// InstallMQTT checks if MQTT is already installed and running, if not then install it from OS repo
func (d *DebOS) InstallMQTT() error {
	cmd := NewCommand("ps aux |awk '/mosquitto/ {print $11}' | awk '/mosquit/ {print}'")
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
func (d *DebOS) IsK8SComponentInstalled(kubeConfig, master string) error {
	return isK8SComponentInstalled(kubeConfig, master)
}

// InstallKubeEdge downloads the provided version of KubeEdge.
// Untar's in the specified location /etc/kubeedge/ and then copies
// the binary to excecutables' path (eg: /usr/local/bin)
func (d *DebOS) InstallKubeEdge(options types.InstallOptions) error {
	return installKubeEdge(options, d.KubeEdgeVersion)
}

// RunEdgeCore starts edgecore with logs being captured
func (d *DebOS) RunEdgeCore() error {
	return runEdgeCore()
}

// KillKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func (d *DebOS) KillKubeEdgeBinary(proc string) error {
	return KillKubeEdgeBinary(proc)
}

// IsKubeEdgeProcessRunning checks if the given process is running or not
func (d *DebOS) IsKubeEdgeProcessRunning(proc string) (bool, error) {
	return IsKubeEdgeProcessRunning(proc)
}
