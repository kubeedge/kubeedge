package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kubeedge/kubeedge/keadm/app/cmd/common"
)

//KubeCloudInstTool embedes Common struct
//It implements ToolsInstaller interface
type KubeCloudInstTool struct {
	Common
}

//InstallTools downloads KubeEdge for the specified version
//and makes the required configuration changes and initiates edgecontroller.
func (cu *KubeCloudInstTool) InstallTools() error {
	cu.SetOSInterface(GetOSInterface())
	cu.SetKubeEdgeVersion(cu.ToolVersion)

	err := cu.InstallKubeEdge()
	if err != nil {
		return err
	}

	err = cu.generateCertificates()
	if err != nil {
		return err
	}

	err = cu.tarCertificates()
	if err != nil {
		return err
	}

	err = cu.StartK8Scluster()
	if err != nil {
		return err
	}

	err = cu.updateManifests()
	if err != nil {
		return err
	}

	//This makes sure the path is created, if it already exists also it is fine
	err = os.MkdirAll(KubeEdgeCloudConfPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeConfPath)
	}

	//Create controller.yaml
	if err = common.WriteControllerYamlFile(KubeEdgeControllerYaml, cu.KubeConfig); err != nil {
		return err
	}

	//Create logger.yaml
	if err = common.WriteCloudLoggingYamlFile(KubeEdgeControllerLoggingYaml); err != nil {
		return err
	}

	//Create modules.yaml
	if err = common.WriteCloudModulesYamlFile(KubeEdgeControllerModulesYaml); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	err = cu.RunEdgeController()
	if err != nil {
		return err
	}
	fmt.Println("Edgecontroller started")

	return nil
}

//generateCertificates - Certifcates ca,cert will be generated in /etc/kubeedge/
func (cu *KubeCloudInstTool) generateCertificates() error {
	//Create certgen.sh
	if err := ioutil.WriteFile(KubeEdgeCloudCertGenPath, CertGenSh, 0775); err != nil {
		return err
	}

	cmd := &Command{Cmd: exec.Command("bash", "-x", KubeEdgeCloudCertGenPath, "genCertAndKey", "edge")}
	err := cmd.ExecuteCmdShowOutput()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", "certificates not installed")
	}
	fmt.Println(stdout)
	fmt.Println("Certificates got generated at:", KubeEdgePath, "ca and", KubeEdgePath, "certs")
	return nil
}

//tarCertificates - certs will be tared at /etc/kubeedge/kubeedge/certificates/certs
func (cu *KubeCloudInstTool) tarCertificates() error {

	tarCmd := fmt.Sprintf("tar -cvzf %s %s", KubeEdgeEdgeCertsTarFileName, strings.Split(KubeEdgeEdgeCertsTarFileName, ".")[0])
	cmd := &Command{Cmd: exec.Command("sh", "-c", tarCmd)}
	cmd.Cmd.Dir = KubeEdgePath
	err := cmd.ExecuteCmdShowOutput()
	stdout := cmd.GetStdOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", "error in tarring the certificates")
	}
	fmt.Println(stdout)
	fmt.Println("Certificates got tared at:", KubeEdgePath, "path, Please copy it to desired edge node (at", KubeEdgePath, "path)")
	return nil
}

//updateManifests - Kubernetes Manifests file will be updated by necessary parameters
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

//RunEdgeController starts edgecontroller process
func (cu *KubeCloudInstTool) RunEdgeController() error {

	filetoCopy := fmt.Sprintf("cp %s/kubeedge/cloud/%s /usr/local/bin/.", KubeEdgePath, KubeCloudBinaryName)
	cmd := &Command{Cmd: exec.Command("sh", "-c", filetoCopy)}
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		fmt.Println("in error")
		return fmt.Errorf("%s", errout)

	}
	binExec := fmt.Sprintf("chmod +x /usr/local/bin/%s && %s > %s/kubeedge/cloud/%s.log 2>&1 &", KubeCloudBinaryName, KubeCloudBinaryName, KubeEdgePath, KubeCloudBinaryName)
	cmd = &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.Cmd.Env = os.Environ()
	env := fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/cloud", KubeEdgePath)
	cmd.Cmd.Env = append(cmd.Cmd.Env, env)
	err = cmd.ExecuteCmdShowOutput()
	errout = cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())
	fmt.Println("KubeEdge controller is running, For logs visit", KubeEdgePath+"kubeedge/cloud/")
	return nil
}

//TearDown method will remove the edge node from api-server and stop edgecontroller process
func (cu *KubeCloudInstTool) TearDown() error {

	cu.SetOSInterface(GetOSInterface())

	//Stops kubeadm
	binExec := fmt.Sprintf("echo 'y' | kubeadm reset &&  rm -rf ~/.kube")
	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	err := cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("kubeadm reset failed %s", errout)
	}

	//Kill edgecontroller process
	cu.KillKubeEdgeBinary(KubeCloudBinaryName)

	return nil
}
