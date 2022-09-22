package util

import (
	"fmt"

	"github.com/blang/semver"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

// PacmanOS struct objects shall have information of the tools version to be installed
// on Hosts having PacmanOS.
// It implements OSTypeInstaller interface
type PacmanOS struct {
	KubeEdgeVersion semver.Version
	IsEdgeNode      bool
}

// SetKubeEdgeVersion sets the KubeEdge version for the objects instance
func (o *PacmanOS) SetKubeEdgeVersion(version semver.Version) {
	o.KubeEdgeVersion = version
}

// InstallMQTT checks if MQTT is already installed and running, if not then install it from OS repo
func (o *PacmanOS) InstallMQTT() error {
	cmd := NewCommand("ps aux |awk '/mosquitto/ {print $11}' | awk '/mosquit/ {print}'")
	if err := cmd.Exec(); err != nil {
		return err
	}

	if stdout := cmd.GetStdOut(); stdout != "" {
		fmt.Println("Host has", stdout, "already installed and running. Hence skipping the installation steps !!!")
		return nil
	}

	// Install mqttInst
	cmd = NewCommand("pacman -Syy --noconfirm mosquitto")
	if err := cmd.Exec(); err != nil {
		return err
	}
	fmt.Println(cmd.GetStdOut())

	fmt.Println("MQTT is installed in this host")

	return nil
}

// IsK8SComponentInstalled checks if said K8S version is already installed in the host
func (o *PacmanOS) IsK8SComponentInstalled(kubeConfig, master string) error {
	return isK8SComponentInstalled(kubeConfig, master)
}

// InstallKubeEdge downloads the provided version of KubeEdge.
// Untar's in the specified location /etc/kubeedge/ and then copies
// the binary to excecutables' path (eg: /usr/local/bin)
func (o *PacmanOS) InstallKubeEdge(options types.InstallOptions) error {
	return installKubeEdge(options, o.KubeEdgeVersion)
}

// RunEdgeCore sets the environment variable GOARCHAIUS_CONFIG_PATH for the configuration path
// and the starts edgecore with logs being captured
func (o *PacmanOS) RunEdgeCore() error {
	return runEdgeCore()
}

// KillKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func (o *PacmanOS) KillKubeEdgeBinary(proc string) error {
	return KillKubeEdgeBinary(proc)
}

// IsKubeEdgeProcessRunning checks if the given process is running or not
func (o *PacmanOS) IsKubeEdgeProcessRunning(proc string) (bool, error) {
	return IsKubeEdgeProcessRunning(proc)
}
