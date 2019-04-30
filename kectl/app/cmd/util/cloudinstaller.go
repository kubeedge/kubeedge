package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"
)

type KubeCloudInstTool struct {
	Common
}

func (cu *KubeCloudInstTool) InstallTools() error {
	cu.SetOSInterface(GetOSInterface())
	cu.SetKubeEdgeVersion(cu.ToolVersion)

	err := cu.InstallKubeEdge()
	if err != nil {

		return err
	}
	fmt.Println("Installation of kubeedge package is sucessfull")
	err = cu.generatecertificates()
	if err != nil {
		fmt.Println(" in err")
		return err
	}
	fmt.Println("Certificates got genertaed and its kept in /etc/kubeedge/kubeedge certificates folder")
	err = cu.tarcertificates()
	if err != nil {
		return err
	}
	fmt.Println("Certificates got Tared and kept /etc/kubeedge/kubeedge/certificates folder Please copy the certificates to the respective edge node")

	err = cu.startkubernetescluster()
	if err != nil {
		return err
	}
	fmt.Println("Kubernetes cluster started")
	err = cu.apiserverHealthcheck()
	if err != nil {
		return err
	}
	fmt.Println("")
	err = cu.updatemanifests()
	if err != nil {
		return err
	}
	fmt.Println("Updation of manifests is sucessful")

	err = cu.updatecontrolleryaml()
	if err != nil {
		return err
	}
	fmt.Println("Updation of controller yaml is sucess")

	time.Sleep(10 * time.Second)
	err = cu.RunEdgeController()
	if err != nil {
		return err
	}
	fmt.Println("Edgecontroller started")

	return nil
}

//Certifcates ca,cert will be generated in /etc/kubeedge/kubeedge/certificates
func (cu *KubeCloudInstTool) generatecertificates() error {
	cmd := &Command{Cmd: exec.Command("bash", "-x", "/etc/kubeedge/kubeedge/tools/certgen.sh", "genCertAndKey", "edge")}
	err := cmd.ExecuteCmdShowOutput()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", "certificates not installed")
		fmt.Println("in error")
	}
	fmt.Println(stdout)
	return nil
}

//certificates tar file will be generated in /etc/kubeedge/kubeedge
func (cu *KubeCloudInstTool) tarcertificates() error {

	cmd := &Command{Cmd: exec.Command("sh", "-c", "tar -cvzf certificates.tar certificates")}
	cmd.Cmd.Dir = KubeEdgeConfigPath + "kubeedge" // or whatever directory it's in
	err := cmd.ExecuteCmdShowOutput()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", "certificates not tared")
		fmt.Println("in error")
	}
	fmt.Println(stdout)
	return nil
}

//controller yaml ca and cert path will be replaces
func (cu *KubeCloudInstTool) updatecontrolleryaml() error {

	filetoReplace := fmt.Sprintf("sed -i 's|ca: .*|ca: %sca/rootCA.crt|g' %s && sed -i 's|cert: .*|cert: %scerts/edge.crt|g' %s && sed -i 's|key: .*|key: %scerts/edge.key|g' %s ", KubeCloudCertificatePath, KubeControllerConfig, KubeCloudCertificatePath, KubeControllerConfig, KubeCloudCertificatePath, KubeControllerConfig)
	cmd := &Command{Cmd: exec.Command("sh", "-c", filetoReplace)}
	//cmd := &Command{Cmd: exec.Command("sh", "-c","sed -i 's|ca: .*|ca: ca/rootCA.crt|g' ",KubeCloudCertificatePath,KubeControllerConfig)}       // or whatever directory it's in
	err := cmd.ExecuteCmdShowOutput()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", "Update controller yaml failed", errout)
		fmt.Println("in error")
	}
	fmt.Println(stdout)
	return nil
}

//kubeadm init will be called and cluster will be started
func (cu *KubeCloudInstTool) startkubernetescluster() error {
	fmt.Println("in start kubernetes cluster")
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
		stdout := cmd.GetStdOutput()
		errout := cmd.GetStdErr()
		if err != nil || errout != "" {
			fmt.Errorf("kubernetes Installation failed:%s", stdout)
		}

		cmd = &Command{Cmd: exec.Command("sh", "-c", "rm  $HOME/.kube/config && mkdir -p $HOME/.kube && echo y |  cp -i /etc/kubernetes/admin.conf $HOME/.kube/config &&  sudo chown $(id -u):$(id -g) $HOME/.kube/config")}
		err = cmd.ExecuteCmdShowOutput()
		stdout = cmd.GetStdOutput()
		errout = cmd.GetStdErr()
		if err != nil || errout != "" {
			fmt.Errorf("kubernetes Installation failed:%s", stdout)
		}
	}
	return nil
}

//Kubernetes Manifests file will be updated by necessary parameters
func (cu *KubeCloudInstTool) updatemanifests() error {

	input, err := ioutil.ReadFile(KubeCloudApiserverYamlPath)
	if err != nil {
		fmt.Println(err)
		return err
	}

	output := bytes.Replace(input, []byte("insecure-port=0"), []byte("insecure-port=8080"), -1)

	if err = ioutil.WriteFile(KubeCloudApiserverYamlPath, output, 0666); err != nil {
		fmt.Println(err)
		return err
	}

	lines, err := file2lines(KubeCloudApiserverYamlPath)
	if err != nil {
		return err
	}

	fileContent := ""
	for i, line := range lines {
		if i == KubeCloudReplaceIndex {
			fileContent += KubeCloudReplaceString
		}
		fileContent += line
		fileContent += "\n"
	}

	ioutil.WriteFile(KubeCloudApiserverYamlPath, []byte(fileContent), 0644)
	return nil

}

func file2lines(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return linesFromReader(f)
}

func linesFromReader(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

//starts the Edgecontroller
func (u *KubeCloudInstTool) RunEdgeController() error {

	filetoCopy := fmt.Sprintf("rm /usr/local/bin/%s && cp %s/kubeedge/cloud/%s /usr/local/bin/.", KubeCloudBinaryName, KubeEdgeConfigPath, KubeCloudBinaryName)
	cmd := &Command{Cmd: exec.Command("sh", "-c", filetoCopy)}
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		fmt.Println("in error")
		return fmt.Errorf("%s", errout)

	}
	binExec := fmt.Sprintf("chmod +x /usr/local/bin/%s && %s > %s/kubeedge/cloud/%s.log 2>&1 &", KubeCloudBinaryName, KubeCloudBinaryName, KubeEdgeConfigPath, KubeCloudBinaryName)
	cmd = &Command{Cmd: exec.Command("sh", "-c", binExec)}
	fmt.Println("binexec is %v", binExec)
	cmd.Cmd.Env = os.Environ()
	env := fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/cloud", KubeEdgeConfigPath)
	cmd.Cmd.Env = append(cmd.Cmd.Env, env)
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())
	return nil
}

//checks the health of Api server
func (cu *KubeCloudInstTool) apiserverHealthcheck() error {

	return nil
}

//kubeadm reset will be called
func (u *KubeCloudInstTool) ResetKubernetes() error {
	binExec := fmt.Sprintf("echo 'y' | kubeadm reset && sudo rm -rf ~/.kube")
	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("kubernetes installation failed %s", errout)
		println("in error")
	}
	return nil
}

//kills edgecontroller
func (u *KubeCloudInstTool) KillEdgeController() error {
	binExec := fmt.Sprintf("kill -9 $(ps aux | grep '[e]%s' | awk '{print $2}') && pkill -9 apiserver", KubeCloudBinaryName[1:])
	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.ExecuteCommand()
	fmt.Println("Edgecontroller is stopped, For logs visit", KubeEdgeConfigPath+"kubeedge/cloud")
	return nil
}
