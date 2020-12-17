package util

import (
	"fmt"
	"os"
	"strings"

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

	//This makes sure the path is created, if it already exists also it is fine
	err = os.MkdirAll(KubeEdgeConfigDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeConfigDir)
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

	if err := types.Write2File(KubeEdgeCloudCoreNewYaml, cloudCoreConfig); err != nil {
		return err
	}

	err = cu.RunCloudCore()
	if err != nil {
		return err
	}
	fmt.Println("CloudCore started")

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
	command = fmt.Sprintf(" %s > %s/%s.log 2>&1 &", KubeCloudBinaryName, KubeEdgeLogPath, KubeCloudBinaryName)

	cmd = NewCommand(command)
	cmd.Cmd.Env = os.Environ()
	env := fmt.Sprintf("GOARCHAIUS_CONFIG_PATH=%skubeedge/cloud", KubeEdgePath)
	cmd.Cmd.Env = append(cmd.Cmd.Env, env)

	if err := cmd.Exec(); err != nil {
		return err
	}

	fmt.Println(cmd.GetStdOut())

	fmt.Println("KubeEdge cloudcore is running, For logs visit: ", KubeEdgeLogPath+KubeCloudBinaryName+".log")

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
