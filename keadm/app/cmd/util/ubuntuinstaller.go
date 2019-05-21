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
	"strings"

	types "github.com/kubeedge/kubeedge/keadm/app/cmd/common"
)

const downloadRetryTimes int = 3

// Ubuntu releases
const (
	UbuntuXenial = "xenial"
	UbuntuBionic = "bionic"
)

//UbuntuOS struct objects shall have information of the tools version to be installed
//on Hosts having Ubuntu OS.
//It implements OSTypeInstaller interface
type UbuntuOS struct {
	DockerVersion     string
	KubernetesVersion string
	KubeEdgeVersion   string
	IsEdgeNode        bool //True - Edgenode False - Cloudnode
}

//SetDockerVersion sets the Docker version for the objects instance
func (u *UbuntuOS) SetDockerVersion(version string) {
	u.DockerVersion = version
}

//SetK8SVersionAndIsNodeFlag sets the K8S version for the objects instance
//It also sets if this host shall act as edge node or not
func (u *UbuntuOS) SetK8SVersionAndIsNodeFlag(version string, flag bool) {
	u.KubernetesVersion = version
	u.IsEdgeNode = flag
}

//SetKubeEdgeVersion sets the KubeEdge version for the objects instance
func (u *UbuntuOS) SetKubeEdgeVersion(version string) {
	u.KubeEdgeVersion = version
}

//IsToolVerInRepo checks if the tool mentioned in available in OS repo or not
func (u *UbuntuOS) IsToolVerInRepo(toolName, version string) (bool, error) {
	//Check if requested Docker or K8S components said version is available in OS repo or not

	chkToolVer := fmt.Sprintf("apt-cache madison '%s' | grep -w %s | head -1 | awk '{$1=$1};1' | cut -d' ' -f 3", toolName, version)
	cmd := &Command{Cmd: exec.Command("sh", "-c", chkToolVer)}
	cmd.ExecuteCommand()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()

	if errout != "" {
		return false, fmt.Errorf("%s", errout)
	}

	if stdout != "" {
		fmt.Println(toolName, stdout, "is available in OS repo")
		return true, nil
	}

	fmt.Println(toolName, "version", version, "not found in OS repo")
	return false, nil
}

func (u *UbuntuOS) addDockerRepositoryAndUpdate() error {
	//lsb_release -cs
	cmd := &Command{Cmd: exec.Command("sh", "-c", "lsb_release -cs")}
	cmd.ExecuteCommand()
	distVersion := cmd.GetStdOutput()
	if distVersion == "" {
		return fmt.Errorf("ubuntu dist version not available")
	}
	fmt.Println("Ubuntu distribution version is", distVersion)

	//'apt-get update'
	cmd = &Command{Cmd: exec.Command("sh", "-c", "apt-get update")}
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	//"curl -fsSL \"$DOWNLOAD_URL/linux/$lsb_dist/gpg\" | apt-key add"
	//Get the GPG key
	curl := fmt.Sprintf("curl -fsSL \"%s/linux/%s/gpg\" | apt-key add", DefaultDownloadURL, UbuntuOSType)
	cmd = &Command{Cmd: exec.Command("sh", "-c", curl)}
	cmd.ExecuteCommand()
	curlOutput := cmd.GetStdOutput()
	if curlOutput == "" {
		return fmt.Errorf("not able add the apt key")
	}
	fmt.Println(curlOutput)

	//Add the repo in OS source.list
	aptRepo := fmt.Sprintf("deb [arch=$(dpkg --print-architecture)] %s/linux/%s %s stable", DefaultDownloadURL, UbuntuOSType, distVersion)
	updtRepo := fmt.Sprintf("echo \"%s\" > /etc/apt/sources.list.d/docker.list", aptRepo)
	cmd = &Command{Cmd: exec.Command("sh", "-c", updtRepo)}
	cmd.ExecuteCommand()
	updtRepoErr := cmd.GetStdErr()
	if updtRepoErr != "" {
		return fmt.Errorf("not able add update repo due to error : %s", updtRepoErr)
	}

	//Do an apt-get update
	cmd = &Command{Cmd: exec.Command("sh", "-c", "apt-get update")}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	return nil
}

//IsDockerInstalled checks if docker is installed in the host or not
func (u *UbuntuOS) IsDockerInstalled(defVersion string) (types.InstallState, error) {
	cmd := &Command{Cmd: exec.Command("sh", "-c", "docker -v | cut -d ' ' -f3 | cut -d ',' -f1")}
	cmd.ExecuteCommand()
	str := cmd.GetStdOutput()

	if strings.Contains(str, u.DockerVersion) {
		return types.AlreadySameVersionExist, nil
	}

	if err := u.addDockerRepositoryAndUpdate(); err != nil {
		return types.VersionNAInRepo, err
	}

	if str == "" {
		return types.NewInstallRequired, nil
	}

	isReqVerAvail, err := u.IsToolVerInRepo("docker-ce", u.DockerVersion)
	if err != nil {
		return types.VersionNAInRepo, err
	}

	var isDefVerAvail bool
	if u.DockerVersion != defVersion {
		isDefVerAvail, err = u.IsToolVerInRepo("docker-ce", defVersion)
		if err != nil {
			return types.VersionNAInRepo, err
		}
	}

	if isReqVerAvail {
		return types.NewInstallRequired, nil
	}

	if isDefVerAvail {
		return types.DefVerInstallRequired, nil
	}

	return types.VersionNAInRepo, nil
}

//InstallDocker will install the specified docker in the host
func (u *UbuntuOS) InstallDocker() error {
	fmt.Println("Installing ", u.DockerVersion, "version of docker")

	//Do an apt-get update
	instPreReq := fmt.Sprintf("apt-get install -y %s", DockerPreqReqList)
	cmd := &Command{Cmd: exec.Command("sh", "-c", instPreReq)}
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	//Get the exact version string from OS repo, so that it can search and install.
	chkDockerVer := fmt.Sprintf("apt-cache madison 'docker-ce' | grep %s | head -1 | awk '{$1=$1};1' | cut -d' ' -f 3", u.DockerVersion)
	cmd = &Command{Cmd: exec.Command("sh", "-c", chkDockerVer)}
	cmd.ExecuteCommand()
	stdout := cmd.GetStdOutput()
	errout = cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}

	fmt.Println("Expected docker version to install is", stdout)

	//Install docker-ce
	dockerInst := fmt.Sprintf("apt-get install -y --allow-change-held-packages --allow-downgrades docker-ce=%s", stdout)
	cmd = &Command{Cmd: exec.Command("sh", "-c", dockerInst)}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	fmt.Println("Docker", u.DockerVersion, "version is installed in this Host")

	return nil
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
	cmd = &Command{Cmd: exec.Command("sh", "-c", mqttInst)}
	err := cmd.ExecuteCmdShowOutput()
	stdout = cmd.GetStdOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	fmt.Println("MQTT is installed in this host")

	return nil
}

//IsK8SComponentInstalled checks if said K8S version is already installed in the host
func (u *UbuntuOS) IsK8SComponentInstalled(component, defVersion string) (types.InstallState, error) {

	find := fmt.Sprintf("dpkg -l | grep %s | awk '{print $3}'", component)
	cmd := &Command{Cmd: exec.Command("sh", "-c", find)}
	cmd.ExecuteCommand()
	str := cmd.GetStdOutput()

	if strings.Contains(str, u.KubernetesVersion) {
		return types.AlreadySameVersionExist, nil
	}

	if err := u.addK8SRepositoryAndUpdate(); err != nil {
		return types.VersionNAInRepo, err
	}

	if str == "" {
		return types.NewInstallRequired, nil
	}

	isReqVerAvail, err := u.IsToolVerInRepo(component, u.KubernetesVersion)
	if err != nil {
		return types.VersionNAInRepo, err
	}

	var isDefVerAvail bool
	if u.KubernetesVersion != defVersion {
		isDefVerAvail, _ = u.IsToolVerInRepo(component, defVersion)
		if err != nil {
			return types.VersionNAInRepo, err
		}
	}

	if isReqVerAvail {
		return types.NewInstallRequired, nil
	}

	if isDefVerAvail {
		return types.DefVerInstallRequired, nil
	}

	return types.VersionNAInRepo, nil
}

func (u *UbuntuOS) addK8SRepositoryAndUpdate() error {
	//Get the distribution version
	cmd := &Command{Cmd: exec.Command("sh", "-c", "lsb_release -cs")}
	cmd.ExecuteCommand()
	distVersion := cmd.GetStdOutput()
	if distVersion == "" {
		return fmt.Errorf("ubuntu dist version not available")
	}
	fmt.Println("Ubuntu distribution version is", distVersion)
	distVersionForSuite := distVersion
	if distVersion == UbuntuBionic {
		// No bionic-specific version is available on apt.kubernetes.io.
		// Use xenial version instead.
		distVersionForSuite = UbuntuXenial
	}
	suite := fmt.Sprintf("kubernetes-%s", distVersionForSuite)
	fmt.Println("Deb suite to use:", suite)

	//Do an apt-get update
	cmd = &Command{Cmd: exec.Command("sh", "-c", "apt-get update")}
	err := cmd.ExecuteCmdShowOutput()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	//curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
	//Get the GPG key
	curl := fmt.Sprintf("curl -s %s | apt-key add -", KubernetesGPGURL)
	cmd = &Command{Cmd: exec.Command("sh", "-c", curl)}
	cmd.ExecuteCommand()
	curlOutput := cmd.GetStdOutput()
	curlErr := cmd.GetStdErr()
	if curlOutput == "" || curlErr != "" {
		return fmt.Errorf("not able add the apt key due to error : %s", curlErr)
	}
	fmt.Println(curlOutput)

	//Add K8S repo to local apt-get source.list
	aptRepo := fmt.Sprintf("deb %s %s main", KubernetesDownloadURL, suite)
	updtRepo := fmt.Sprintf("echo \"%s\" > /etc/apt/sources.list.d/kubernetes.list", aptRepo)
	cmd = &Command{Cmd: exec.Command("sh", "-c", updtRepo)}
	cmd.ExecuteCommand()
	updtRepoErr := cmd.GetStdErr()
	if updtRepoErr != "" {
		return fmt.Errorf("not able add update repo due to error : %s", updtRepoErr)
	}

	//Do an apt-get update
	cmd = &Command{Cmd: exec.Command("sh", "-c", "apt-get update")}
	err = cmd.ExecuteCmdShowOutput()
	stdout = cmd.GetStdOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)
	return nil
}

//InstallK8S will install kubeadm, kudectl and kubelet for the cloud node
func (u *UbuntuOS) InstallK8S() error {
	k8sComponent := "kubeadm"
	fmt.Println("Installing", k8sComponent, u.KubernetesVersion, "version")

	//Get the exact version string from OS repo, so that it can search and install.
	chkKubeadmVer := fmt.Sprintf("apt-cache madison '%s' | grep %s | head -1 | awk '{$1=$1};1' | cut -d' ' -f 3", k8sComponent, u.KubernetesVersion)
	cmd := &Command{Cmd: exec.Command("sh", "-c", chkKubeadmVer)}
	cmd.ExecuteCommand()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}

	fmt.Println("Expected K8S('", k8sComponent, "') version to install is", stdout)

	//Install respective K8S components based on where it is being installed
	k8sInst := fmt.Sprintf("apt-get install -y --allow-change-held-packages --allow-downgrades kubeadm=%s kubelet=%s kubectl=%s", stdout, stdout, stdout)
	cmd = &Command{Cmd: exec.Command("sh", "-c", k8sInst)}
	err := cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	fmt.Println(k8sComponent, "version", u.KubernetesVersion, "is installed in this Host")

	return nil
}

//StartK8Scluster will do "kubeadm init" and cluster will be started
func (u *UbuntuOS) StartK8Scluster() error {
	var install bool
	cmd := &Command{Cmd: exec.Command("sh", "-c", "kubeadm version")}
	cmd.ExecuteCommand()
	str := cmd.GetStdOutput()
	if str != "" {
		install = true
	} else {
		install = false
	}
	if install == true {
		cmd := &Command{Cmd: exec.Command("sh", "-c", "swapoff -a && kubeadm init")}
		err := cmd.ExecuteCmdShowOutput()
		errout := cmd.GetStdErr()
		if err != nil || errout != "" {
			return fmt.Errorf("kubeadm init failed:%s", errout)
		}

		fmt.Println(cmd.GetStdOutput())

		cmd = &Command{Cmd: exec.Command("sh", "-c", " mkdir -p $HOME/.kube && cp -r /etc/kubernetes/admin.conf $HOME/.kube/config &&  sudo chown $(id -u):$(id -g) $HOME/.kube/config")}
		err = cmd.ExecuteCmdShowOutput()
		errout = cmd.GetStdErr()
		if err != nil || errout != "" {
			return fmt.Errorf("copying configuration file of kubeadm failed:%s", errout)
		}
		fmt.Println(cmd.GetStdOutput())
	} else {
		return fmt.Errorf("kubeadm not installed in this host")
	}
	fmt.Println("Kubeadm init successfully executed")
	return nil
}

//InstallKubeEdge downloads the provided version of KubeEdge.
//Untar's in the specified location /etc/kubeedge/ and then copies
//the binary to /usr/local/bin path.
func (u *UbuntuOS) InstallKubeEdge() error {
	var (
		dwnldURL string
		cmd      *Command
	)

	err := os.MkdirAll(KubeEdgePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgePath)
	}

	cmd = &Command{Cmd: exec.Command("sh", "-c", "dpkg --print-architecture")}
	cmd.ExecuteCommand()
	arch := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}

	//Check if the same version exists, then skip the download and just untar and continue
	//TODO: It is always better to have the checksum validation of the downloaded file
	//and checksum available at download URL. So that both can be compared to see if
	//proper download has happened and then only proceed further.
	//Currently it is missing and once checksum is in place, checksum check required
	//to be added here.
	filename := fmt.Sprintf("kubeedge-v%s-linux-%s.tar.gz", u.KubeEdgeVersion, arch)
	checksumFilename := fmt.Sprintf("checksum_kubeedge-v%s-linux-%s.txt", u.KubeEdgeVersion, arch)
	filePath := fmt.Sprintf("%s%s", KubeEdgePath, filename)
	fileStat, err := os.Stat(filePath)
	if err == nil && fileStat.Name() != "" {
		fmt.Println("Expected or Default KubeEdge version", u.KubeEdgeVersion, "is already downloaded")
		goto SKIPDOWNLOADAND
	}

	for i := 0; i < downloadRetryTimes; i++ {
		//Download the tar from repo
		dwnldURL = fmt.Sprintf("cd %s && wget -k --no-check-certificate --progress=bar:force %s/v%s/%s", KubeEdgePath, KubeEdgeDownloadURL, u.KubeEdgeVersion, filename)
		cmd = &Command{Cmd: exec.Command("sh", "-c", dwnldURL)}
		if err := cmd.ExecuteCmdShowOutput(); err != nil {
			return err
		}
		if errout := cmd.GetStdErr(); errout != "" {
			return fmt.Errorf("%s", errout)
		}
		fmt.Println()

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
			cmd = &Command{Cmd: exec.Command("sh", "-c", cmdStr)}
			if err := cmd.ExecuteCmdShowOutput(); err != nil {
				return err
			}
			if errout := cmd.GetStdErr(); errout != "" {
				return fmt.Errorf("%s", errout)
			}
		} else {
			return fmt.Errorf("failed to verify the checksum of %s", filename)
		}
	}

SKIPDOWNLOADAND:
	untarFileAndMove := fmt.Sprintf("cd %s && tar -C %s -xvzf %s && cp %s/kubeedge/edge/%s /usr/local/bin/.", KubeEdgePath, KubeEdgePath, filename, KubeEdgePath, KubeEdgeBinaryName)
	cmd = &Command{Cmd: exec.Command("sh", "-c", untarFileAndMove)}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	return nil
}

//RunEdgeCore sets the environment variable GOARCHAIUS_CONFIG_PATH for the configuration path
//and the starts edge_core with logs being captured
func (u *UbuntuOS) RunEdgeCore() error {
	binExec := fmt.Sprintf("chmod +x /usr/local/bin/%s && %s > %s/kubeedge/edge/%s.log 2>&1 &", KubeEdgeBinaryName, KubeEdgeBinaryName, KubeEdgePath, KubeEdgeBinaryName)
	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.Cmd.Env = os.Environ()
	env := fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/edge", KubeEdgePath)
	cmd.Cmd.Env = append(cmd.Cmd.Env, env)
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())
	fmt.Println("KubeEdge edge core is running, For logs visit", KubeEdgePath+"kubeedge/edge/")
	return nil
}

//KillKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func (u *UbuntuOS) KillKubeEdgeBinary(proc string) error {
	binExec := fmt.Sprintf("kill -9 $(ps aux | grep '[%s]%s' | awk '{print $2}')", proc[0:1], proc[1:])
	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.ExecuteCommand()
	fmt.Println("KubeEdge is stopped, For logs visit", KubeEdgePath+"kubeedge/edge/")
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
