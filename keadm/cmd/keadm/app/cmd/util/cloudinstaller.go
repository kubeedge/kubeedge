package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/blang/semver"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

//KubeCloudInstTool embedes Common struct
//It implements ToolsInstaller interface
type KubeCloudInstTool struct {
	Common
	AdvertiseAddress string
	DNSName          string
	TarballPath      string
}

// InstallTools downloads KubeEdge for the specified version
// and makes the required configuration changes and initiates cloudcore.
func (cu *KubeCloudInstTool) InstallTools() error {
	cu.SetOSInterface(GetOSInterface())
	cu.SetKubeEdgeVersion(cu.ToolVersion)

	opts := &types.InstallOptions{
		TarballPath:   cu.TarballPath,
		ComponentType: types.CloudCore,
	}

	err := cu.InstallKubeEdge(*opts)
	if err != nil {
		return err
	}
	if cu.ToolVersion.LT(semver.MustParse("1.3.0")) {
		err = cu.generateCertificates()
		if err != nil {
			return err
		}

		err = cu.tarCertificates()
		if err != nil {
			return err
		}
	}

	if cu.ToolVersion.GE(semver.MustParse("1.2.0")) {
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

		if cu.AdvertiseAddress != "" {
			cloudCoreConfig.Modules.CloudHub.AdvertiseAddress = strings.Split(cu.AdvertiseAddress, ",")
		}

		if cu.DNSName != "" {
			cloudCoreConfig.Modules.CloudHub.DNSNames = strings.Split(cu.DNSName, ",")
		}

		if cu.ToolVersion.Major == 1 && cu.ToolVersion.Minor == 2 {
			cloudCoreConfig.Modules.CloudHub.TLSPrivateKeyFile = KubeEdgeCloudDefaultCertPath + "server.key"
			cloudCoreConfig.Modules.CloudHub.TLSCertFile = KubeEdgeCloudDefaultCertPath + "server.crt"
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

	command := fmt.Sprintf("%s genCertAndKey server", KubeEdgeCloudCertGenPath)
	cmd := NewCommand(command)
	if err := cmd.Exec(); err != nil {
		return err
	}

	fmt.Println(cmd.GetStdOut())
	fmt.Println("Certificates got generated at:", KubeEdgePath, "ca and", KubeEdgePath, "certs")
	return nil
}

//tarCertificates - certs will be tared at /etc/kubeedge/kubeedge/certificates/certs
func (cu *KubeCloudInstTool) tarCertificates() error {
	tarCmd := fmt.Sprintf("tar -cvzf %s %s", KubeEdgeEdgeCertsTarFileName, strings.Split(KubeEdgeEdgeCertsTarFileName, ".")[0])
	cmd := NewCommand(tarCmd)
	cmd.Cmd.Dir = KubeEdgePath
	if err := cmd.Exec(); err != nil {
		return err
	}

	fmt.Println(cmd.GetStdOut())
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
	cmd := NewCommand(command)
	if err := cmd.Exec(); err != nil {
		return err
	}

	// start cloudcore
	if cu.ToolVersion.GE(semver.MustParse("1.1.0")) {
		command = fmt.Sprintf(" %s > %s/%s.log 2>&1 &", KubeCloudBinaryName, KubeEdgeLogPath, KubeCloudBinaryName)
	} else {
		command = fmt.Sprintf("%s > %skubeedge/cloud/%s.log 2>&1 &", KubeCloudBinaryName, KubeEdgePath, KubeCloudBinaryName)
	}

	cmd = NewCommand(command)
	cmd.Cmd.Env = os.Environ()
	env := fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/cloud", KubeEdgePath)
	cmd.Cmd.Env = append(cmd.Cmd.Env, env)

	if err := cmd.Exec(); err != nil {
		return err
	}

	fmt.Println(cmd.GetStdOut())

	if cu.ToolVersion.GE(semver.MustParse("1.1.0")) {
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
	if err := cu.KillKubeEdgeBinary(KubeCloudBinaryName); err != nil {
		return err
	}
	// clean kubeedge namespace
	err := cu.cleanNameSpace("kubeedge", cu.KubeConfig)
	if err != nil {
		return fmt.Errorf("fail to clean kubeedge namespace, err:%v", err)
	}
	return nil
}
