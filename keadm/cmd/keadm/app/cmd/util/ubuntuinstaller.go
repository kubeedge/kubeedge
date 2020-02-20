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
	"os"
	"os/exec"
	"strconv"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

const downloadRetryTimes int = 3

//UbuntuOS struct objects shall have information of the tools version to be installed
//on Hosts having Ubuntu OS.
//It implements OSTypeInstaller interface
type UbuntuOS struct {
	KubeEdgeVersion string
	IsEdgeNode      bool //True - Edgenode False - Cloudnode
}

//SetKubeEdgeVersion sets the KubeEdge version for the objects instance
func (u *UbuntuOS) SetKubeEdgeVersion(version string) {
	u.KubeEdgeVersion = version
}

//InstallMQTT checks if MQTT is already installed and running, if not then install it from OS repo
func (u *UbuntuOS) InstallMQTT() error {
	mqttRunning := fmt.Sprintf("ps aux |awk '/mosquitto/ {print $1}' | awk '/mosquit/ {print}'")
	cmd := &Command{Cmd: exec.Command("sh", "-c", mqttRunning)}
	cmd.ExecuteCommand()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}
	if stdout != "" {
		fmt.Println("Host has", stdout, "already installed and running. Hence skipping the installation steps !!!")
		return nil
	}

	//Install mqttInst
	mqttInst := fmt.Sprintf("apt-get install -y --allow-change-held-packages --allow-downgrades mosquitto")
	stdout, err := runCommandWithShell(mqttInst)
	if err != nil {
		return err
	}
	fmt.Println(stdout)

	fmt.Println("MQTT is installed in this host")

	return nil
}

// IsK8SComponentInstalled checks if said K8S version is already installed in the host
func (u *UbuntuOS) IsK8SComponentInstalled(kubeConfig, master string) error {
	config, err := BuildConfig(kubeConfig, master)
	if err != nil {
		return fmt.Errorf("Failed to build config, err: %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to init discovery client, err: %v", err)
	}

	discoveryClient.RESTClient().Post()
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return fmt.Errorf("Failed to get the version of K8s master, please check whether K8s was successfully installed, err: %v", err)
	}

	if serverVersion.GitVersion == "" {
		return fmt.Errorf("Failed to get the version of K8s master, please check whether K8s was successfully installed, err: %v", err)
	}

	k8sMinorVersion, _ := strconv.Atoi(serverVersion.Minor)
	if k8sMinorVersion >= types.DefaultK8SMinimumVersion {
		return nil
	}

	return fmt.Errorf("Your minor version of K8s is lower than %d, please reinstall newer version", types.DefaultK8SMinimumVersion)

}

// InstallKubeEdge downloads the provided version of KubeEdge.
// Untar's in the specified location /etc/kubeedge/ and then copies
// the binary to excecutables' path (eg: /usr/local/bin)
func (u *UbuntuOS) InstallKubeEdge() error {
	var (
		dwnldURL string
		cmd      *Command
	)

	err := os.MkdirAll(KubeEdgePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgePath)
	}

	arch, err := getSystemArch()
	if err != nil {
		return err
	}

	//Check if the same version exists, then skip the download and just untar and continue
	//TODO: It is always better to have the checksum validation of the downloaded file
	//and checksum available at download URL. So that both can be compared to see if
	//proper download has happened and then only proceed further.
	//Currently it is missing and once checksum is in place, checksum check required
	//to be added here.
	dirname := fmt.Sprintf("kubeedge-v%s-linux-%s", u.KubeEdgeVersion, arch)
	filename := fmt.Sprintf("kubeedge-v%s-linux-%s.tar.gz", u.KubeEdgeVersion, arch)
	checksumFilename := fmt.Sprintf("checksum_kubeedge-v%s-linux-%s.tar.gz.txt", u.KubeEdgeVersion, arch)
	filePath := fmt.Sprintf("%s%s", KubeEdgePath, filename)
	fileStat, err := os.Stat(filePath)
	if err == nil && fileStat.Name() != "" {
		fmt.Println("Expected or Default KubeEdge version", u.KubeEdgeVersion, "is already downloaded")
		goto SKIPDOWNLOADAND
	}

	for i := 0; i < downloadRetryTimes; i++ {
		//Download the tar from repo
		dwnldURL = fmt.Sprintf("cd %s && wget -k --no-check-certificate --progress=bar:force %s/v%s/%s", KubeEdgePath, KubeEdgeDownloadURL, u.KubeEdgeVersion, filename)
		_, err := runCommandWithShell(dwnldURL)
		if err != nil {
			return err
		}

		//Verify the tar with checksum
		fmt.Printf("%s checksum: \n", filename)
		cmdStr := fmt.Sprintf("cd %s && sha512sum %s | awk '{split($0,a,\"[ ]\"); print a[1]}'", KubeEdgePath, filename)
		cmd = &Command{Cmd: exec.Command("sh", "-c", cmdStr)}
		cmd.ExecuteCommand()
		desiredChecksum := cmd.GetStdOutput()
		fmt.Printf("%s \n\n", cmd.GetStdOutput())

		fmt.Printf("%s content: \n", checksumFilename)
		cmdStr = fmt.Sprintf("wget -qO- %s/v%s/%s", KubeEdgeDownloadURL, u.KubeEdgeVersion, checksumFilename)
		cmd = &Command{Cmd: exec.Command("sh", "-c", cmdStr)}
		cmd.ExecuteCommand()
		actualChecksum := cmd.GetStdOutput()
		fmt.Printf("%s \n\n", cmd.GetStdOutput())

		if desiredChecksum == actualChecksum {
			break
		}

		if i < downloadRetryTimes-1 {
			fmt.Printf("Failed to verify the checksum of %s, try to download it again ... \n\n", filename)
			//Cleanup the downloaded files
			cmdStr = fmt.Sprintf("cd %s && rm -f %s", KubeEdgePath, filename)
			_, err := runCommandWithShell(cmdStr)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("failed to verify the checksum of %s", filename)
		}
	}

SKIPDOWNLOADAND:
	// Compatible with 1.0.0
	var untarFileAndMoveEdgeCore, moveCloudCore string
	if u.KubeEdgeVersion >= "1.1.0" {
		untarFileAndMoveEdgeCore = fmt.Sprintf("cd %s && tar -C %s -xvzf %s && cp %s%s/edge/%s %s/",
			KubeEdgePath, KubeEdgePath, filename, KubeEdgePath, dirname, KubeEdgeBinaryName, KubeEdgeUsrBinPath)
		moveCloudCore = fmt.Sprintf("cd %s && cp %s%s/cloud/cloudcore/%s %s/",
			KubeEdgePath, KubeEdgePath, dirname, KubeCloudBinaryName, KubeEdgeUsrBinPath)
	} else {
		untarFileAndMoveEdgeCore = fmt.Sprintf("cd %s && tar -C %s -xvzf %s && cp %skubeedge/edge/%s %s/.",
			KubeEdgePath, KubeEdgePath, filename, KubeEdgePath, KubeEdgeBinaryName, KubeEdgeUsrBinPath)
		moveCloudCore = fmt.Sprintf("cd %s && cp %skubeedge/cloud/%s %s/.",
			KubeEdgePath, KubeEdgePath, KubeCloudBinaryName, KubeEdgeUsrBinPath)
	}

	stdout, err := runCommandWithShell(untarFileAndMoveEdgeCore)
	if err != nil {
		return err
	}
	fmt.Println(stdout)

	stdout, err = runCommandWithShell(moveCloudCore)
	if err != nil {
		return err
	}
	fmt.Println(stdout)

	return nil
}

//RunEdgeCore sets the environment variable GOARCHAIUS_CONFIG_PATH for the configuration path
//and the starts edgecore with logs being captured
func (u *UbuntuOS) RunEdgeCore() error {
	// create the log dir for kubeedge
	err := os.MkdirAll(KubeEdgeLogPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeLogPath)
	}

	var binExec string
	if u.KubeEdgeVersion >= "1.1.0" {
		binExec = fmt.Sprintf("chmod +x %s/%s && %s > %s/%s.log 2>&1 &",
			KubeEdgeUsrBinPath, KubeEdgeBinaryName, KubeEdgeBinaryName, KubeEdgeLogPath, KubeEdgeBinaryName)
	} else {
		binExec = fmt.Sprintf("chmod +x %s/%s && %s > %skubeedge/edge/%s.log 2>&1 &",
			KubeEdgeUsrBinPath, KubeEdgeBinaryName, KubeEdgeBinaryName, KubeEdgePath, KubeEdgeBinaryName)
	}

	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.Cmd.Env = os.Environ()
	env := fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/edge", KubeEdgePath)
	cmd.Cmd.Env = append(cmd.Cmd.Env, env)
	err = cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	if u.KubeEdgeVersion >= "1.1.0" {
		fmt.Println("KubeEdge edgecore is running, For logs visit: ", KubeEdgeLogPath+KubeEdgeBinaryName+".log")
	} else {
		fmt.Println("KubeEdge edgecore is running, For logs visit", KubeEdgePath, "kubeedge/edge/")
	}

	return nil
}

//KillKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func (u *UbuntuOS) KillKubeEdgeBinary(proc string) error {
	binExec := fmt.Sprintf("kill -9 $(ps aux | grep '[%s]%s' | awk '{print $2}')", proc[0:1], proc[1:])
	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.ExecuteCommand()

	if u.KubeEdgeVersion >= "1.1.0" {
		fmt.Println("KubeEdge", proc, "is stopped, For logs visit: ", KubeEdgeLogPath)
	} else {
		fmt.Println("KubeEdge is stopped, For logs visit", KubeEdgePath+"kubeedge/edge/")
	}

	return nil
}

//IsKubeEdgeProcessRunning checks if the given process is running or not
func (u *UbuntuOS) IsKubeEdgeProcessRunning(proc string) (bool, error) {
	procRunning := fmt.Sprintf("ps aux | grep '[%s]%s' | awk '{print $2}'", proc[0:1], proc[1:])
	cmd := &Command{Cmd: exec.Command("sh", "-c", procRunning)}
	cmd.ExecuteCommand()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if errout != "" {
		return false, fmt.Errorf("%s", errout)
	}
	if stdout != "" {
		return true, nil
	}
	return false, nil
}

// runCommandWithShell executes the given command with "sh -c".
// It returns an error if the command outputs anything on the stderr.
func runCommandWithShell(command string) (string, error) {
	cmd := &Command{Cmd: exec.Command("sh", "-c", command)}
	err := cmd.ExecuteCmdShowOutput()
	if err != nil {
		return "", err
	}
	errout := cmd.GetStdErr()
	if errout != "" {
		return "", fmt.Errorf("%s", errout)
	}
	return cmd.GetStdOutput(), nil
}

// build Config from flags
func BuildConfig(kubeConfig, master string) (conf *rest.Config, err error) {
	config, err := clientcmd.BuildConfigFromFlags(master, kubeConfig)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func getSystemArch() (string, error) {
	cmd := &Command{Cmd: exec.Command("sh", "-c", "dpkg --print-architecture")}
	cmd.ExecuteCommand()
	arch := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if errout != "" {
		return "", fmt.Errorf("%s", errout)
	}
	return arch, nil
}
