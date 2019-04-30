package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

//ResetKubernetes will call Kubeadm reset
func (u *KubeCloudInstTool) ResetKubernetes() error {

	binExec := fmt.Sprintf("echo 'y' | kubeadm reset &&  rm -rf ~/.kube")
	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("kubernetes reset failed %s", errout)
	}
	return nil
}

// KillEdgeController forcefully kills the EdgeController process
func (u *KubeCloudInstTool) KillEdgeController() error {

	binExec := fmt.Sprintf("kill -9 $(ps aux | grep '[e]%s' | awk '{print $2}') && pkill -9 apiserver", KubeCloudBinaryName[1:])
	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.ExecuteCommand()
	fmt.Println("Edgecontroller is stopped, For logs visit", KubeEdgePath+"kubeedge/cloud")
	return nil
}
