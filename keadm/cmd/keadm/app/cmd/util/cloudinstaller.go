package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
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

	err := cu.InstallKubeEdge(types.CloudCore)
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

		cloudCoreConfig := v1alpha1.NewDefaultCloudCoreConfig()
		if cu.KubeConfig != "" {
			cloudCoreConfig.KubeAPIConfig.KubeConfig = cu.KubeConfig
		}

		if cu.Master != "" {
			cloudCoreConfig.KubeAPIConfig.Master = cu.Master
		}

		if err := types.Write2File(KubeEdgeCloudCoreNewYaml, cloudCoreConfig); err != nil {
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
		if err = types.WriteControllerYamlFile(KubeEdgeCloudCoreYaml, cu.KubeConfig); err != nil {
			return err
		}

		//Create modules.yaml
		if err = types.WriteCloudModulesYamlFile(KubeEdgeCloudCoreModulesYaml); err != nil {
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

	cmd := &Command{Cmd: exec.Command("bash", "-x", KubeEdgeCloudCertGenPath, "genCertAndKey", "server")}
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

	// add +x for cloudcore
	command := fmt.Sprintf("chmod +x %s/%s", KubeEdgeUsrBinPath, KubeCloudBinaryName)
	if _, err := runCommandWithShell(command); err != nil {
		return err
	}

	// start cloudcore
	if cu.ToolVersion >= "1.1.0" {
		command = fmt.Sprintf(" %s > %s/%s.log 2>&1 &", KubeCloudBinaryName, KubeEdgeLogPath, KubeCloudBinaryName)
	} else {
		command = fmt.Sprintf("%s > %skubeedge/cloud/%s.log 2>&1 &", KubeCloudBinaryName, KubeEdgePath, KubeCloudBinaryName)
	}
	cmd := &Command{Cmd: exec.Command("sh", "-c", command)}
	cmd.Cmd.Env = os.Environ()
	env := fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/cloud", KubeEdgePath)
	cmd.Cmd.Env = append(cmd.Cmd.Env, env)
	cmd.ExecuteCommand()
	if errout := cmd.GetStdErr(); errout != "" {
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
	cu.SetKubeEdgeVersion(cu.ToolVersion)

	//Kill cloudcore process
	cu.KillKubeEdgeBinary(KubeCloudBinaryName)

	return nil
}
