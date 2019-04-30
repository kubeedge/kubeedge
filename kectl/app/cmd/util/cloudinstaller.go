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
	err = cu.generateCertificates()
	if err != nil {
		fmt.Println(" in err")
		return err
	}
	fmt.Println("Certificates got generated at : /etc/kubeedge/")
	err = cu.updateControlleryaml()
	if err != nil {
		return err
	}
	fmt.Println("Certificates got Updated at : /etc/kubeedge/kubeedge/cloud/conf/controller.yaml")

	err = cu.tarCertificates()
	if err != nil {
		return err
	}
	fmt.Println("Certificates got Tared at : /etc/kubeedge/kubeedge/certificates folder, Please copy the certificates to the respective edge node")

	err = cu.startKubernetescluster()
	if err != nil {
		return err
	}
	fmt.Println("Kubernetes cluster started")

	err = cu.updateManifests()
	if err != nil {
		return err
	}
	fmt.Println("Updation of manifests is sucessful")

	err = cu.updateControlleryaml()
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

//generateCertificates  generates ca,cert in /etc/kubeedge
func (cu *KubeCloudInstTool) generateCertificates() error {
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

//updateControlleryaml replaces certificate path in Controller.yaml
func (cu *KubeCloudInstTool) updateControlleryaml() error {

	filetoReplace := fmt.Sprintf("sed -i 's|ca: .*|ca: %sca/rootCA.crt|g' %s && sed -i 's|cert: .*|cert: %scerts/edge.crt|g' %s && sed -i 's|key: .*|key: %scerts/edge.key|g' %s ", KubeCloudCertificatePath, KubeControllerConfig, KubeCloudCertificatePath, KubeControllerConfig, KubeCloudCertificatePath, KubeControllerConfig)
	cmd := &Command{Cmd: exec.Command("sh", "-c", filetoReplace)}
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

//tarCertificates,certs will be tared at /etc/kubeedge
func (cu *KubeCloudInstTool) tarCertificates() error {

	cmd := &Command{Cmd: exec.Command("sh", "-c", "tar -cvzf certs.tar edge.crt edge.key && cp -r certs.tar /etc/kubeedge")}
	cmd.Cmd.Dir = KubeEdgePath + "/certs"

	err := cmd.ExecuteCmdShowOutput()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", "certificates not tared")
	}
	fmt.Println(stdout)
	return nil
}

//startKubernetescluster checks kubeadm version and calls kubeadm init
func (cu *KubeCloudInstTool) startKubernetescluster() error {
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

		cmd = &Command{Cmd: exec.Command("sh", "-c", " mkdir -p $HOME/.kube && cp -r /etc/kubernetes/admin.conf $HOME/.kube/config &&  sudo chown $(id -u):$(id -g) $HOME/.kube/config")}
		err = cmd.ExecuteCmdShowOutput()
		stdout = cmd.GetStdOutput()
		errout = cmd.GetStdErr()
		if err != nil || errout != "" {
			fmt.Errorf("kubernetes Installation failed:%s", stdout)
		}
	}
	return nil
}

//updateManifests updates Kubernetes Manifests file by necessary parameters
func (cu *KubeCloudInstTool) updateManifests() error {
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

//RunEdgeController starts the Edgecontroller
func (u *KubeCloudInstTool) RunEdgeController() error {

	filetoCopy := fmt.Sprintf(" cp %s/kubeedge/cloud/%s /usr/local/bin/.", KubeEdgePath, KubeCloudBinaryName)
	cmd := &Command{Cmd: exec.Command("sh", "-c", filetoCopy)}
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		fmt.Println("in error")
		return fmt.Errorf("%s", errout)

	}
	binExec := fmt.Sprintf("chmod +x /usr/local/bin/%s && %s > %s/kubeedge/cloud/%s.log 2>&1 &", KubeCloudBinaryName, KubeCloudBinaryName, KubeEdgePath, KubeCloudBinaryName)
	cmd = &Command{Cmd: exec.Command("sh", "-c", binExec)}
	fmt.Println("binexec is %v", binExec)
	cmd.Cmd.Env = os.Environ()
	env := fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/cloud", KubeEdgePath)
	cmd.Cmd.Env = append(cmd.Cmd.Env, env)
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())
	return nil
}

func (ku *KubeCloudInstTool) TearDown() error {
	ku.SetOSInterface(GetOSInterface())
	ku.KillEdgeController()
	ku.ResetKubernetes()

	return nil
}
