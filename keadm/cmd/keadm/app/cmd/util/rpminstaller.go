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
	"strings"

	"github.com/blang/semver"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

const (
	openEulerVendorName = "openEuler"
)

// RpmOS struct objects shall have information of the tools version to be installed
// on Hosts having RpmOS OS.
// It implements OSTypeInstaller interface
type RpmOS struct {
	KubeEdgeVersion semver.Version
	IsEdgeNode      bool //True - Edgenode False - Cloudnode
}

// SetKubeEdgeVersion sets the KubeEdge version for the objects instance
func (r *RpmOS) SetKubeEdgeVersion(version semver.Version) {
	r.KubeEdgeVersion = version
}

// InstallMQTT checks if MQTT is already installed and running, if not then install it from OS repo
// Information is used from https://www.digitalocean.com/community/tutorials/how-to-install-and-secure-the-mosquitto-mqtt-messaging-broker-on-centos-7
func (r *RpmOS) InstallMQTT() error {
	// check MQTT
	cmd := NewCommand("ps aux |awk '/mosquitto/ {print $11}' | awk '/mosquit/ {print}'")
	if err := cmd.Exec(); err != nil {
		return err
	}

	if stdout := cmd.GetStdOut(); stdout != "" {
		fmt.Println("Host has", stdout, "already installed and running. Hence skipping the installation steps !!!")
		return nil
	}

	commands := []string{
		"yum -y install epel-release",
		"yum -y install mosquitto",
		"systemctl start mosquitto",
		"systemctl enable mosquitto",
	}

	vendorName, err := getOSVendorName()
	if err != nil {
		fmt.Printf("Get OS vendor name failed: %v\n", err)
	}
	// epel-release package does not included in openEuler
	if vendorName == openEulerVendorName {
		commands = commands[1:]
	}

	// install MQTT
	for _, command := range commands {
		cmd := NewCommand(command)
		if err := cmd.Exec(); err != nil {
			return err
		}
	}
	fmt.Println("install MQTT service successfully.")

	return nil
}

// IsK8SComponentInstalled checks if said K8S version is already installed in the host
func (r *RpmOS) IsK8SComponentInstalled(kubeConfig, master string) error {
	return isK8SComponentInstalled(kubeConfig, master)
}

// InstallKubeEdge downloads the provided version of KubeEdge.
// Untar's in the specified location /etc/kubeedge/ and then copies
// the binary to excecutables' path (eg: /usr/local/bin)
func (r *RpmOS) InstallKubeEdge(options types.InstallOptions) error {
	return installKubeEdge(options, r.KubeEdgeVersion)
}

// RunEdgeCore starts edgecore with logs being captured
func (r *RpmOS) RunEdgeCore() error {
	return runEdgeCore()
}

// KillKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func (r *RpmOS) KillKubeEdgeBinary(proc string) error {
	return KillKubeEdgeBinary(proc)
}

// IsKubeEdgeProcessRunning checks if the given process is running or not
func (r *RpmOS) IsKubeEdgeProcessRunning(proc string) (bool, error) {
	return IsKubeEdgeProcessRunning(proc)
}

func getOSVendorName() (string, error) {
	cmd := NewCommand("cat /etc/os-release | grep -E \"^NAME=\" | awk -F'=' '{print $2}'")
	if err := cmd.Exec(); err != nil {
		return "", err
	}
	vendor := strings.Trim(cmd.GetStdOut(), "\"")

	return vendor, nil
}
