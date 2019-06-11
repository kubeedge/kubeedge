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

func (C *CentOS) addDockerRepositoryAndUpdate() error {
	// cmd := &Command{Cmd: exec.Command("sh", "-c", "lsb_release -cs")}
	// cmd.ExecuteCommand()
	// distVersion := cmd.GetStdOutput()
	// if distVersion == "" {
	// 	return fmt.Errorf("CentOS dist version not available")
	// }
	// fmt.Println("CentOS distribution version is", distVersion)
	cmd := &Command{Cmd: exec.Command("sh", "-c", "yum install -y yum-utils")}
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())
	fmt.Println("utils installed")

	cmd = &Command{Cmd: exec.Command("sh", "-c", "yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo")}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())
	fmt.Println("docker-ce.repo installed")

	cmd = &Command{Cmd: exec.Command("sh", "-c", "yum makecache")}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	return nil
}

//IsDockerInstalled checks if docker is installed in the host or not
func (c *CentOS) IsDockerInstalled(defVersion string) (types.InstallState, error) {

	//yum list installed | grep docker-ce | awk '{print $2}' | cut -d'-' -f 1
	cmd := &Command{Cmd: exec.Command("sh", "-c", "yum list installed | grep docker-ce | awk '{print $2}' | cut -d'-' -f 1")}
	cmd.ExecuteCommand()
	fmt.Println("IsDockerInstalled", cmd.GetStdOutput())
	str := cmd.GetStdOutput()
	if str == "" {
		return types.NewInstallRequired, nil
	}

	if strings.Contains(cmd.GetStdOutput(), c.DockerVersion) {
		return types.AlreadySameVersionExist, nil
	}

	if err := c.addDockerRepositoryAndUpdate(); err != nil {
		return types.VersionNAInRepo, err
	}

	isReqVerAvail, err := c.IsToolVerInRepo("docker-ce", c.DockerVersion)
	if err != nil {
		return types.VersionNAInRepo, err
	}

	var isDefVerAvail bool
	if c.DockerVersion != defVersion {
		isDefVerAvail, err = c.IsToolVerInRepo("docker-ce", defVersion)
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

	//yum list installed | grep docker-ce | awk '{print $2}' | cut -d'-' -f 1
	//18.06.1.ce

	return types.VersionNAInRepo, nil
}

//InstallDocker will install the specified docker in the host
func (c *CentOS) InstallDocker() error {
	fmt.Println("InstallDocker called")
	fmt.Println("Installing", c.DockerVersion, "Version of docker")

	if err := c.addDockerRepositoryAndUpdate(); err != nil {
		return err
	}
	//Do an yum update
	/*	instPreReq := fmt.Sprintf("yum install -y %s", DockerPreqReqList)
		cmd := &Command{Cmd: exec.Command("sh", "-c", instPreReq)}
		err := cmd.ExecuteCmdShowOutput()
		errout := cmd.GetStdErr()
		if err != nil || errout != "" {
			return fmt.Errorf("%s", errout)
		}
		fmt.Println(cmd.GetStdOutput())*/
	chkDockerVer := fmt.Sprintf(" yum list --showduplicates 'docker-ce' | grep %s| head -1 | awk '{print $2}'", c.DockerVersion)
	cmd := &Command{Cmd: exec.Command("sh", "-c", chkDockerVer)}
	cmd.ExecuteCommand()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}

	//Install docker-ce
	fmt.Println("stdout is %s", stdout)
	dockerInst := fmt.Sprintf("sudo yum install -y  --skip-broken docker-ce-%s", stdout)
	cmd = &Command{Cmd: exec.Command("sh", "-c", dockerInst)}
	err := cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	fmt.Println("Docker", c.DockerVersion, "version is installed in this Host")
	dockerstart := "sudo systemctl start docker"
	cmd = &Command{Cmd: exec.Command("sh", "-c", dockerstart)}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	fmt.Println("Docker", c.DockerVersion, "version is started")

	return nil
	// yum install -y yum-utils
	//yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
	//	yum-config-manager --add-repo /etc/yum.repos.d/docker.repo
	// yum makecache
	// yum list --showduplicates 'docker-ce' | grep '17.06.0' | head -1 | awk '{print $2}'
	// yum install -y docker-ce-17.06.0.ce-1.el7.centos
	// [root@localhost ~]# systemctl start docker
	// [root@localhost ~]# ---> Always restart  systemctl restart docker
	// [root@localhost ~]#
	// IF downgrade yum downgrade -y docker-ce-17.06.0.ce-1.el7.centos
	// Check always for version, if it is a downgrade or upgrade

}

//IsToolVerInRepo checks if the tool mentioned in available in OS repo or not
func (c *CentOS) IsToolVerInRepo(toolName, version string) (bool, error) {
	chkToolVer := fmt.Sprintf(" yum list --showduplicates '%s' | grep '%s' | head -1 | awk '{print $2}'", toolName, version)
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

	fmt.Println("IsToolVerInRepo called")
	fmt.Println(toolName, "version", version, "not found in OS repo")
	return false, nil
	//yum --cacheonly list | grep openssl
	//For K8S, dont check in repo, just install
}

//InstallMQTT checks if MQTT is already installed and running, if not then install it from OS repo
//Information is used from https://www.digitalocean.com/community/tutorials/how-to-install-and-secure-the-mosquitto-mqtt-messaging-broker-on-centos-7
func (c *CentOS) InstallMQTT() error {

	//yum -y install epel-release
	cmd := &Command{Cmd: exec.Command("sh", "-c", "yum -y install epel-release")}
	err := cmd.ExecuteCmdShowOutput()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	//yum -y install mosquitto
	cmd = &Command{Cmd: exec.Command("sh", "-c", "yum -y install mosquitto")}
	err = cmd.ExecuteCmdShowOutput()
	stdout = cmd.GetStdOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	//systemctl start mosquitto
	cmd = &Command{Cmd: exec.Command("sh", "-c", "systemctl start mosquitto")}
	cmd.ExecuteCommand()
	stdout = cmd.GetStdOutput()
	errout = cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	//systemctl enable mosquitto
	cmd = &Command{Cmd: exec.Command("sh", "-c", "systemctl enable mosquitto")}
	cmd.ExecuteCommand()
	stdout = cmd.GetStdOutput()
	errout = cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)

	return nil
}

//IsK8SComponentInstalled checks if said K8S version is already installed in the host
func (c *CentOS) IsK8SComponentInstalled(component, defVersion string) (types.InstallState, error) {
	// 	[root@localhost ~]# yum list installed | grep kubeadm | awk '{print $2}' | cut -d'-' -f 1
	// 1.14.1
	// [root@localhost ~]#
	// [root@localhost ~]# yum list installed | grep kubeadm
	// kubeadm.x86_64                          1.14.1-0                       @kubernetes
	// [root@localhost ~]#

	find := fmt.Sprintf("yum list installed | grep %s | awk '{print $2}' | cut -d'-' -f 1", component)
	cmd := &Command{Cmd: exec.Command("sh", "-c", find)}
	cmd.ExecuteCommand()
	str := cmd.GetStdOutput()
	if str == "" {
		return types.NewInstallRequired, nil
	}

	if strings.Contains(cmd.GetStdOutput(), c.KubernetesVersion) {
		return types.AlreadySameVersionExist, nil
	}

	if err := c.addK8SRepositoryAndUpdate(); err != nil {
		return types.VersionNAInRepo, err
	}

	isReqVerAvail, err := c.IsToolVerInRepo(component, c.KubernetesVersion)
	if err != nil {
		return types.VersionNAInRepo, err
	}

	var isDefVerAvail bool
	if c.KubernetesVersion != defVersion {
		isDefVerAvail, _ = c.IsToolVerInRepo(component, defVersion)
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

func (c *CentOS) addK8SRepositoryAndUpdate() error {
	//Get the distribution version
	// cmd := &Command{Cmd: exec.Command("sh", "-c", "lsb_release -cs")}
	// cmd.ExecuteCommand()
	// distVersion := cmd.GetStdOutput()
	// if distVersion == "" {
	// 	return fmt.Errorf("CentOS dist version not available")
	// }
	// fmt.Println("CentOS distribution version is", distVersion)

	//Do an apt-get update
	// cmd = &Command{Cmd: exec.Command("sh", "-c", "apt-get update")}
	// err := cmd.ExecuteCmdShowOutput()
	// stdout := cmd.GetStdOutput()
	// errout := cmd.GetStdErr()
	// if err != nil || errout != "" {
	// 	return fmt.Errorf("%s", errout)
	// }
	// fmt.Println(stdout)

	//curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
	//Get the GPG key
	// curl := fmt.Sprintf("curl -s %s | apt-key add -", KubernetesGPGURL)
	// cmd = &Command{Cmd: exec.Command("sh", "-c", curl)}
	// cmd.ExecuteCommand()
	// curlOutput := cmd.GetStdOutput()
	// curlErr := cmd.GetStdErr()
	// if curlOutput == "" || curlErr != "" {
	// 	return fmt.Errorf("not able add the apt key due to error : %s", curlErr)
	// }
	// fmt.Println(curlOutput)

	//Add K8S repo to local apt-get source.list
	aptRepo := fmt.Sprintf("[kubernetes]\nname=Kubernetes\nbaseurl=%s\nenabled=1\ngpgcheck=1\nrepo_gpgcheck=1\ngpgkey=%s\nexclude=kube*\n", KubernetesBaseurl, KubernetesGpgkey)
	updtRepo := fmt.Sprintf("echo \"%s\" > /etc/yum.repos.d/kubernetes.repo", aptRepo)
	cmd := &Command{Cmd: exec.Command("sh", "-c", updtRepo)}
	cmd.ExecuteCommand()
	updtRepoErr := cmd.GetStdErr()
	fmt.Printf("updtRepoErr %s", updtRepoErr)
	if updtRepoErr != "" {
		return fmt.Errorf("not able add update repo due to error : %s", updtRepoErr)
	}

	//Do an apt-get update
	// 	# Set SELinux in permissive mode (effectively disabling it)
	// setenforce 0
	// sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config
	cmd = &Command{Cmd: exec.Command("sh", "-c", "setenforce 0 && sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config")}
	err := cmd.ExecuteCmdShowOutput()
	fmt.Println("addK8SRepositoryAndUpdat4 ")
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(stdout)
	return nil
}

//InstallK8S will install kubeadm, kudectl and kubelet for the cloud node
func (c *CentOS) InstallK8S() error {
	//fmt.Println("InstallK8S called")
	//Follow https://kubernetes.io/docs/setup/independent/install-kubeadm/
	// yum install -y kubelet kubeadm kubectl --disableexcludes=kubernetes
	// systemctl enable --now kubelet
	k8sComponent := "kubeadm"
	fmt.Println("Installing", k8sComponent, c.KubernetesVersion, "version")
	if err := c.addK8SRepositoryAndUpdate(); err != nil {
		return err
	}

	//Get the exact version string from OS repo, so that it can search and install.
	chkKubeadmVer := fmt.Sprintf("yum list --showduplicates --disableexcludes=kubernetes %s| grep %s |awk '{print $2}' | cut -d'-' -f 1", k8sComponent, c.KubernetesVersion)
	cmd := &Command{Cmd: exec.Command("sh", "-c", chkKubeadmVer)}
	cmd.ExecuteCommand()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if errout != "" {
		return fmt.Errorf("%s", errout)
	}

	fmt.Println("Expected K8S('", k8sComponent, "') version to install is", stdout)

	//Install respective K8S components based on where it is being installed
	k8sInst := fmt.Sprintf("yum install -y kubeadm-%s kubelet-%s kubectl-%s --disableexcludes=kubernetes", stdout, stdout, stdout)
	cmd = &Command{Cmd: exec.Command("sh", "-c", k8sInst)}
	err := cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	enaKubelet := fmt.Sprintf("systemctl enable --now kubelet")
	cmd = &Command{Cmd: exec.Command("sh", "-c", enaKubelet)}
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	fmt.Println(k8sComponent, "version", c.KubernetesVersion, "is installed in this Host")

	return nil
}

//StartK8Scluster will do "kubeadm init" and cluster will be started
func (c *CentOS) StartK8Scluster() error {
	fmt.Println("StartK8sCluster")
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
		cmd := &Command{Cmd: exec.Command("sh", "-c", "swapoff -a && kubeadm init --pod-network-cidr 10.244.0.0/16")}
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
func (c *CentOS) InstallKubeEdge() error {
	var (
		dwnldURL string
		cmd      *Command
	)

	err := os.MkdirAll(KubeEdgePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgePath)
	}

	//cmd = &Command{Cmd: exec.Command("sh", "-c", "dpkg --print-architecture")}
	cmd = &Command{Cmd: exec.Command("sh", "-c", "arch")}
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
	if arch == "x86_64" {
		arch = "amd64"
	}
	filename := fmt.Sprintf("kubeedge-v%s-linux-%s.tar.gz", c.KubeEdgeVersion, arch)
	checksumFilename := fmt.Sprintf("checksum_kubeedge-v%s-linux-%s.txt", c.KubeEdgeVersion, arch)
	filePath := fmt.Sprintf("%s%s", KubeEdgePath, filename)
	fileStat, err := os.Stat(filePath)
	if err == nil && fileStat.Name() != "" {
		fmt.Println("Expected or Default KubeEdge version", c.KubeEdgeVersion, "is already downloaded")
		goto SKIPDOWNLOADAND
	}

	for i := 0; i < downloadRetryTimes; i++ {
		//Download the tar from repo
		dwnldURL = fmt.Sprintf("cd %s && wget -k --no-check-certificate --progress=bar:force %s/v%s/%s", KubeEdgePath, KubeEdgeDownloadURL, c.KubeEdgeVersion, filename)
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
		cmdStr = fmt.Sprintf("wget -qO- %s/v%s/%s", KubeEdgeDownloadURL, c.KubeEdgeVersion, checksumFilename)
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
func (c *CentOS) RunEdgeCore() error {
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
func (c *CentOS) KillKubeEdgeBinary(proc string) error {
	binExec := fmt.Sprintf("kill -9 $(ps aux | grep '[%s]%s' | awk '{print $2}')", proc[0:1], proc[1:])
	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.ExecuteCommand()
	fmt.Println("KubeEdge is stopped, For logs visit", KubeEdgePath+"kubeedge/edge/")
	return nil
}

//IsKubeEdgeProcessRunning checks if the given process is running or not
func (c *CentOS) IsKubeEdgeProcessRunning(proc string) (bool, error) {
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
