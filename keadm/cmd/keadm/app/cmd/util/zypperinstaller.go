package util

import (
	"fmt"

	"github.com/blang/semver"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/util/execs"
)

// ZypperOS provides package-manager-specific operations for hosts using Zypper.
// It implements the OSTypeInstaller interface.
type ZypperOS struct {
	KubeEdgeVersion semver.Version
	IsEdgeNode      bool
}

// SetKubeEdgeVersion sets the KubeEdge version for the objects instance
func (o *ZypperOS) SetKubeEdgeVersion(version semver.Version) {
	o.KubeEdgeVersion = version
}

// InstallMQTT checks if MQTT is already installed and running, if not then install it from OS repo
func (o *ZypperOS) InstallMQTT() error {
	cmd := execs.NewCommand("ps aux |awk '/mosquitto/ {print $11}' | awk '/mosquit/ {print}'")
	if err := cmd.Exec(); err != nil {
		return err
	}

	if stdout := cmd.GetStdOut(); stdout != "" {
		fmt.Println("Host has", stdout, "already installed and running. Hence skipping the installation steps !!!")
		return nil
	}

	// Install mqttInst
	cmd = execs.NewCommand("zypper --non-interactive install mosquitto")
	if err := cmd.Exec(); err != nil {
		return err
	}
	fmt.Println(cmd.GetStdOut())

	fmt.Println("MQTT is installed in this host")

	return nil
}

// IsK8SComponentInstalled checks if said K8S version is already installed in the host
func (o *ZypperOS) IsK8SComponentInstalled(kubeConfig, master string) error {
	return isK8SComponentInstalled(kubeConfig, master)
}

// InstallKubeEdge downloads the provided version of KubeEdge.
// Untar's in the specified location /etc/kubeedge/ and then copies
// the binary to excecutables' path (eg: /usr/local/bin)
func (o *ZypperOS) InstallKubeEdge(options types.InstallOptions) error {
	return installKubeEdge(options, o.KubeEdgeVersion)
}

// RunEdgeCore sets the environment variable GOARCHAIUS_CONFIG_PATH for the configuration path
// and the starts edgecore with logs being captured
func (o *ZypperOS) RunEdgeCore() error {
	return runEdgeCore()
}

// KillKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func (o *ZypperOS) KillKubeEdgeBinary(proc string) error {
	return KillKubeEdgeBinary(proc)
}

// IsKubeEdgeProcessRunning checks if the given process is running or not
func (o *ZypperOS) IsKubeEdgeProcessRunning(proc string) (bool, error) {
	return IsKubeEdgeProcessRunning(proc)
}
