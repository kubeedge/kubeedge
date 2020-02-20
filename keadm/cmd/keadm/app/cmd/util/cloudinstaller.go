package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

//KubeCloudInstTool embedes Common struct
//It implements ToolsInstaller interface
type KubeCloudInstTool struct {
	Common
}

// InstallTools downloads KubeEdge for the specified version
// and makes the required configuration changes and initiates cloudcore.
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

	if cu.ToolVersion >= "1.2.0" {
		//This makes sure the path is created, if it already exists also it is fine
		err = os.MkdirAll(KubeEdgeNewConfigDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("not able to create %s folder path", KubeEdgeNewConfigDir)
		}

		binExec := fmt.Sprintf("chmod +x %s/%s && %s --defaultconfig",
			KubeEdgeUsrBinPath, KubeCloudBinaryName, KubeCloudBinaryName)

		cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
		cmd.ExecuteCommand()
		config := cmd.GetStdOutput()
		errout := cmd.GetStdErr()
		if errout != "" {
			return fmt.Errorf("%s", errout)
		}

		if err = ioutil.WriteFile(KubeEdgeCloudCoreNewYaml, []byte(config), 0666); err != nil {
			return err
		}
	} else {
		//This makes sure the path is created, if it already exists also it is fine
		err = os.MkdirAll(KubeEdgeCloudConfPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("not able to create %s folder path", KubeEdgeConfPath)
		}

		//KubeEdgeCloudCoreYaml:= fmt.Sprintf("%s%s/edge/%s",KubeEdgePath)
		//	KubeEdgePath, KubeEdgePath, filename, KubeEdgePath, dirname, KubeEdgeBinaryName, KubeEdgeUsrBinPath)
		//Create controller.yaml
		if err = common.WriteControllerYamlFile(KubeEdgeCloudCoreYaml, cu.KubeConfig); err != nil {
			return err
		}

		//Create modules.yaml
		if err = common.WriteCloudModulesYamlFile(KubeEdgeCloudCoreModulesYaml); err != nil {
			return err
		}
	}

	time.Sleep(1 * time.Second)

	err = cu.RunCloudCore()
	if err != nil {
		return err
	}
	fmt.Println("CloudCore started")

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

//RunCloudCore starts cloudcore process
func (cu *KubeCloudInstTool) RunCloudCore() error {
	// create the log dir for kubeedge
	err := os.MkdirAll(KubeEdgeLogPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeLogPath)
	}

	var binExec string
	if cu.ToolVersion >= "1.1.0" {
		binExec = fmt.Sprintf("chmod +x %s/%s && %s > %s/%s.log 2>&1 &",
			KubeEdgeUsrBinPath, KubeCloudBinaryName, KubeCloudBinaryName, KubeEdgeLogPath, KubeCloudBinaryName)
	} else {
		binExec = fmt.Sprintf("chmod +x %s/%s && %s > %skubeedge/cloud/%s.log 2>&1 &",
			KubeEdgeUsrBinPath, KubeCloudBinaryName, KubeCloudBinaryName, KubeEdgePath, KubeCloudBinaryName)
	}

	cmd := &Command{Cmd: exec.Command("sh", "-c", binExec)}
	cmd.Cmd.Env = os.Environ()
	env := fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/cloud", KubeEdgePath)
	cmd.Cmd.Env = append(cmd.Cmd.Env, env)
	err = cmd.ExecuteCmdShowOutput()
	errout := cmd.GetStdErr()
	if err != nil || errout != "" {
		return fmt.Errorf("%s", errout)
	}
	fmt.Println(cmd.GetStdOutput())

	if cu.ToolVersion >= "1.1.0" {
		fmt.Println("KubeEdge cloudcore is running, For logs visit: ", KubeEdgeLogPath+KubeCloudBinaryName+".log")
	} else {
		fmt.Println("KubeEdge cloudcore is running, For logs visit", KubeEdgePath+"kubeedge/cloud/")
	}

	return nil
}

//TearDown method will remove the edge node from api-server and stop cloudcore process
func (cu *KubeCloudInstTool) TearDown() error {
	cu.SetOSInterface(GetOSInterface())

	//Kill cloudcore process
	cu.KillKubeEdgeBinary(KubeCloudBinaryName)

	return nil
}
