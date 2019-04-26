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
)

type UbuntuOS struct {
	DockerVersion     string
	KubernetesVersion string
	KubeEdgeVersion   string
	IsEdgeNode        bool //True - Edgenode False - Cloudnode
}

func (u *UbuntuOS) SetDockerVersion(version string) {
	u.DockerVersion = version
}

func (u *UbuntuOS) SetK8SVersionAndIsNodeFlag(version string, flag bool) {
	u.KubernetesVersion = version
	u.IsEdgeNode = flag
}

func (u *UbuntuOS) SetKubeEdgeVersion(version string) {
	u.KubeEdgeVersion = version
}

func (u *UbuntuOS) IsToolVerInRepo(toolName, version string) (bool, error) {
	//Check if requested Docker version available in repo

	//apt-cache madison '$toolname' | grep $version | head -1 | awk '{$1=$1};1' | cut -d' ' -f 3
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

	// fmt.Println("Docker version", u.DockerVersion, "is not available in OS repo")
	// fmt.Println("Docker versions available in OS repo are:")
	// cmd1 := &Command{Cmd: exec.Command("sh", "-c", "apt-cache madison 'docker-ce'")}
	// cmd1.ExecuteCommand()
	// fmt.Println(cmd1.GetStdOutput())
	fmt.Println(toolName, "version", version, "not found in OS repo")
	return false, nil
}

func (u *UbuntuOS) IsDockerInstalled(defVersion string) (InstallState, error) {
	cmd := &Command{Cmd: exec.Command("sh", "-c", "docker -v | cut -d ' ' -f3 | cut -d ',' -f1")}
	cmd.ExecuteCommand()
	str := cmd.GetStdOutput()
	if str == "" {
		return NewInstallRequired, nil
	}

	if strings.Contains(cmd.GetStdOutput(), u.DockerVersion) {
		return AlreadySameVersionExist, nil
	}

	isReqVerAvail, err := u.IsToolVerInRepo("docker-ce", u.DockerVersion)
	if err != nil {
		return VersionNAInRepo, err
	}

	var isDefVerAvail bool
	if u.DockerVersion != defVersion {
		isDefVerAvail, err = u.IsToolVerInRepo("docker-ce", defVersion)
		if err != nil {
			return VersionNAInRepo, err
		}
	}

	if isReqVerAvail {
		return NewInstallRequired, nil
	}

	if isDefVerAvail {
		return DefVerInstallRequired, nil
	}

	return VersionNAInRepo, nil
}

func (u *UbuntuOS) InstallDocker() error {
	fmt.Println("InstallDocker called")

	fmt.Println("Installing ", u.DockerVersion, "version of docker")

	//lsb_release -cs
	cmd := &Command{Cmd: exec.Command("sh", "-c", "lsb_release -cs")}
	cmd.ExecuteCommand()
	distVersion := cmd.GetStdOutput()
	if distVersion == "" {
		return fmt.Errorf("Ubuntu dist version not available")
	}
	fmt.Println("Ubuntu distribution version is", distVersion)

	//'apt-get update -qq >/dev/null'
	cmd = &Command{Cmd: exec.Command("sh", "-c", "apt-get update")}
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	//"apt-get install -y -qq $pre_reqs >/dev/null"
	instPreReq := fmt.Sprintf("apt-get install -y %s", DockerPreqReqList)
	cmd = &Command{Cmd: exec.Command("sh", "-c", instPreReq)}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	// //"curl -fsSL \"$DOWNLOAD_URL/linux/$lsb_dist/gpg\" | apt-key add -qq - >/dev/null"
	curl := fmt.Sprintf("curl -fsSL \"%s/linux/%s/gpg\" | apt-key add", DefaultDownloadURL, UbuntuOSType)
	cmd = &Command{Cmd: exec.Command("sh", "-c", curl)}
	cmd.ExecuteCommand()
	curlOutput := cmd.GetStdOutput()
	if curlOutput == "" {
		return fmt.Errorf("not able add the apt key")
	}
	fmt.Println(curlOutput)

	//apt_repo="deb [arch=$(dpkg --print-architecture)] $DOWNLOAD_URL/linux/$lsb_dist $dist_version $CHANNEL"
	aptRepo := fmt.Sprintf("deb [arch=$(dpkg --print-architecture)] %s/linux/%s %s stable", DefaultDownloadURL, UbuntuOSType, distVersion)
	//"echo \"$apt_repo\" > /etc/apt/sources.list.d/docker.list"
	updtRepo := fmt.Sprintf("echo \"%s\" > /etc/apt/sources.list.d/docker.list", aptRepo)
	cmd = &Command{Cmd: exec.Command("sh", "-c", updtRepo)}
	cmd.ExecuteCommand()
	updtRepoErr := cmd.GetStdErr()
	if updtRepoErr != "" {
		return fmt.Errorf("not able add update repo due to error : %s", updtRepoErr)
	}

	//'apt-get update -qq >/dev/null'
	cmd = &Command{Cmd: exec.Command("sh", "-c", "apt-get update")}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	//apt-cache madison 'docker-ce' | grep $version | head -1 | awk '{$1=$1};1' | cut -d' ' -f 3
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

	fmt.Println("Docker version", u.DockerVersion, "is installed in this Host")

	return nil
}

func (u *UbuntuOS) InstallMQTT() error {

	//Check if MQTT is already installed and running
	//ps aux |awk '/mosquitto/ {print $1}' | awk '/mosquit/ {print}'

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

func (u *UbuntuOS) IsK8SComponentInstalled(component, defVersion string) (InstallState, error) {

	find := fmt.Sprintf("dpkg -l | grep %s | awk '{print $3}'", component)
	cmd := &Command{Cmd: exec.Command("sh", "-c", find)}
	cmd.ExecuteCommand()
	str := cmd.GetStdOutput()
	if str == "" {
		return NewInstallRequired, nil
	}

	if strings.Contains(cmd.GetStdOutput(), u.KubernetesVersion) {
		return AlreadySameVersionExist, nil
	}

	isReqVerAvail, err := u.IsToolVerInRepo(component, u.KubernetesVersion)
	if err != nil {
		return VersionNAInRepo, err
	}

	var isDefVerAvail bool
	if u.KubernetesVersion != defVersion {
		isDefVerAvail, _ = u.IsToolVerInRepo(component, defVersion)
		if err != nil {
			return VersionNAInRepo, err
		}
	}

	if isReqVerAvail {
		return NewInstallRequired, nil
	}

	if isDefVerAvail {
		return DefVerInstallRequired, nil
	}

	return VersionNAInRepo, nil
}

func (u *UbuntuOS) InstallK8S() error {
	fmt.Println("InstallK8S called")

	kcomp := fmt.Sprintf("Installing %s version of ", u.KubernetesVersion)
	if u.IsEdgeNode == true {
		kcomp = kcomp + "kubectl"
	} else {
		kcomp = kcomp + "kubeadm"
	}
	fmt.Println(kcomp)

	//lsb_release -cs
	cmd := &Command{Cmd: exec.Command("sh", "-c", "lsb_release -cs")}
	cmd.ExecuteCommand()
	distVersion := cmd.GetStdOutput()
	if distVersion == "" {
		return fmt.Errorf("Ubuntu dist version not available")
	}
	fmt.Println("Ubuntu distribution version is", distVersion)

	//'apt-get update -qq >/dev/null'
	cmd = &Command{Cmd: exec.Command("sh", "-c", "apt-get update")}
	err := cmd.ExecuteCmdShowOutput()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	//curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
	curl := fmt.Sprintf("curl -s %s | apt-key add -", KubernetesGPGURL)
	cmd = &Command{Cmd: exec.Command("sh", "-c", curl)}
	cmd.ExecuteCommand()
	curlOutput := cmd.GetStdOutput()
	curlErr := cmd.GetStdErr()
	if curlOutput == "" || curlErr != "" {
		return fmt.Errorf("not able add the apt key due to error : %s", curlErr)
	}
	fmt.Println(curlOutput)

	// 	cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
	// deb https://apt.kubernetes.io/ kubernetes-xenial main
	// EOF
	aptRepo := fmt.Sprintf("deb %s kubernetes-%s main", KubernetesDownloadURL, distVersion)
	//"echo \"$apt_repo\" > /etc/apt/sources.list.d/docker.list"
	updtRepo := fmt.Sprintf("echo \"%s\" > /etc/apt/sources.list.d/kubernetes.list", aptRepo)
	cmd = &Command{Cmd: exec.Command("sh", "-c", updtRepo)}
	cmd.ExecuteCommand()
	updtRepoErr := cmd.GetStdErr()
	if updtRepoErr != "" {
		return fmt.Errorf("not able add update repo due to error : %s", updtRepoErr)
	}

	//'apt-get update -qq >/dev/null'
	cmd = &Command{Cmd: exec.Command("sh", "-c", "apt-get update")}
	err = cmd.ExecuteCmdShowOutput()
	stdout = cmd.GetStdOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	k8sComponent := "kubeadm"
	if u.IsEdgeNode == true {
		k8sComponent = "kubectl"
	}
	//apt-cache madison 'kubeadm' | grep $version | head -1 | awk '{$1=$1};1' | cut -d' ' -f 3
	chkKubeadmVer := fmt.Sprintf("apt-cache madison '%s' | grep %s | head -1 | awk '{$1=$1};1' | cut -d' ' -f 3", k8sComponent, u.KubernetesVersion)
	cmd = &Command{Cmd: exec.Command("sh", "-c", chkKubeadmVer)}
	cmd.ExecuteCommand()
	stdout = cmd.GetStdOutput()
	errout = cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}

	fmt.Println("Expected K8S('", k8sComponent, "') version to install is", stdout)

	//Install K8S
	k8sInst := fmt.Sprintf("apt-get install -y --allow-change-held-packages --allow-downgrades kubeadm=%s kubelet=%s kubectl=%s", stdout, stdout, stdout)
	if u.IsEdgeNode == true {
		k8sInst = fmt.Sprintf("apt-get install -y --allow-change-held-packages --allow-downgrades kubectl=%s", stdout)
	}
	cmd = &Command{Cmd: exec.Command("sh", "-c", k8sInst)}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	fmt.Println(k8sComponent, "version", u.KubernetesVersion, "is installed in this Host")

	return nil
}

func (u *UbuntuOS) InstallKubeEdge() error {
	fmt.Println("InstallKubeEdge called")

	var (
		confPath, dwnldURL string
		cmd                *Command
	)

	//Create the default path if not exist.
	src, err := os.Stat(KubeEdgeConfigPath)
	if err == nil && src.IsDir() {
		fmt.Println(KubeEdgeConfigPath, "is available")
		goto DOWNLOADBINARY
	}

	confPath = fmt.Sprintf("mkdir %s", KubeEdgeConfigPath)
	cmd = &Command{Cmd: exec.Command("sh", "-c", confPath)}
	cmd.ExecuteCommand()
	fmt.Println("KubeEdge config path", KubeEdgeConfigPath, "is available")

DOWNLOADBINARY:

	cmd = &Command{Cmd: exec.Command("sh", "-c", "dpkg --print-architecture")}
	cmd.ExecuteCommand()
	arch := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}

	filename := fmt.Sprintf("kubeedge-v%s-linux-%s.tar.gz", u.KubeEdgeVersion, arch)
	filePath := fmt.Sprintf("%s%s", KubeEdgeConfigPath, filename)
	fileStat, err := os.Stat(filePath)
	if err == nil && fileStat.Name() != "" {
		fmt.Println("Expected or Default KubeEdge version", u.KubeEdgeVersion, "is already installed")
		goto SKIPDOWNLOADAND
		//return fmt.Errorf("%s", KubeEdgeVersionAlreadyInstalled)
	}

	//Download the tar for repo
	//filename := fmt.Sprintf("kubeedge-v%s-linux-$(dpkg --print-architecture).tar.gz", u.KubeEdgeVersion)
	dwnldURL = fmt.Sprintf("cd %s && wget -k --no-check-certificate --progress=bar:force %s/v%s/%s", KubeEdgeConfigPath, KubeEdgeDownloadURL, u.KubeEdgeVersion, filename)
	cmd = &Command{Cmd: exec.Command("sh", "-c", dwnldURL)}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

SKIPDOWNLOADAND:
	//tar -C /usr/local -xzf go1.12.4.linux-amd64.tar.gz
	//kubeFolderName := strings.Split(filename, ".")[0]
	untarFileAndMove := fmt.Sprintf("cd %s && tar -C %s -xvzf %s && cp %s/kubeedge/edge/%s /usr/local/bin/.", KubeEdgeConfigPath, KubeEdgeConfigPath, filename, KubeEdgeConfigPath, KubeEdgeBinaryName)
	cmd = &Command{Cmd: exec.Command("sh", "-c", untarFileAndMove)}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	return nil
}

func (u *UbuntuOS) RunEdgeCore() error {

	//Execute edge_core
	binExec := fmt.Sprintf("chmod +x /usr/local/bin/%s && %s > %s/kubeedge/edge/%s.log 2>&1 &", KubeEdgeBinaryName, KubeEdgeBinaryName, KubeEdgeConfigPath, KubeEdgeBinaryName)
	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.Cmd.Env = os.Environ()
	env := fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/edge", KubeEdgeConfigPath)
	cmd.Cmd.Env = append(cmd.Cmd.Env, env)
	//cmd.ExecuteCommand()
        err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())
	fmt.Println("KubeEdge started running, For logs visit", KubeEdgeConfigPath+"edge/")
	return nil
}

func (u *UbuntuOS) KillEdgeCore() error {

	//Execute edge_core
	binExec := fmt.Sprintf("kill -9 $(ps aux | grep '[e]%s' | awk '{print $2}')", KubeEdgeBinaryName[1:])
	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.ExecuteCommand()
	fmt.Println("KubeEdge is stopped, For logs visit", KubeEdgeConfigPath+"edge/")
	return nil
}
