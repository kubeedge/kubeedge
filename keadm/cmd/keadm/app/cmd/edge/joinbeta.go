/*
Copyright 2022 The KubeEdge Authors.

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

package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1/validation"
	pkgutil "github.com/kubeedge/kubeedge/pkg/util"
)

var (
	edgeJoinBetaDescription = `
"keadm beta join" command bootstraps KubeEdge's worker node (at the edge) component.
It will also connect with cloud component to receive
further instructions and forward telemetry data from
devices to cloud
`
	edgeJoinBetaExample = `
keadm beta join --cloudcore-ipport=<ip:port address> --edgenode-name=<unique string as edge identifier>

  - For this command --cloudcore-ipport flag is a required option
  - This command will download and install the default version of pre-requisites and KubeEdge

keadm beta join --cloudcore-ipport=10.20.30.40:10000 --edgenode-name=testing123 --kubeedge-version=v` + common.DefaultKubeEdgeVersion + `
`
)

func NewJoinBetaCommand() *cobra.Command {
	joinOptions := newOption()
	step := common.NewStep()
	cmd := &cobra.Command{
		Use:          "join",
		Short:        "Bootstraps edge component. Checks and install (if required) the pre-requisites. Execute it on any edge node machine you wish to join",
		Long:         edgeJoinBetaDescription,
		Example:      edgeJoinBetaExample,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			step.Printf("Check KubeEdge edgecore process status")
			running, err := util.IsKubeEdgeProcessRunning(util.KubeEdgeBinaryName)
			if err != nil {
				klog.Exitf("Check KubeEdge edgecore process status failed: %v", err)
			}
			if running {
				klog.Exitln("EdgeCore is already running on this node, please run reset to clean up first")
			}

			step.Printf("Check if the management directory is clean")
			if _, err := os.Stat(util.KubeEdgePath); err != nil {
				if os.IsNotExist(err) {
					return
				}
				klog.Exitf("Stat management directory %s failed: %v", util.KubeEdgePath, err)
			}
			entries, err := os.ReadDir(util.KubeEdgePath)
			if err != nil {
				klog.Exitf("Read management directory %s failed: %v", util.KubeEdgePath, err)
			}
			if len(entries) > 0 {
				klog.Exitf("The management directory %s is not clean, please remove it first", util.KubeEdgePath)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			ver, err := util.GetCurrentVersion(joinOptions.KubeEdgeVersion)
			if err != nil {
				klog.Errorf("Edge node join failed: %v", err)
				os.Exit(1)
			}
			joinOptions.KubeEdgeVersion = ver

			if err := join(joinOptions, step); err != nil {
				klog.Errorf("Edge node join failed: %v", err)
				os.Exit(1)
			}
		},
	}

	addJoinOtherFlags(cmd, joinOptions)
	return cmd
}

func newOption() *common.JoinOptions {
	joinOptions := &common.JoinOptions{}
	joinOptions.WithMQTT = true
	joinOptions.CGroupDriver = v1alpha1.CGroupDriverCGroupFS
	joinOptions.CertPath = common.DefaultCertPath
	joinOptions.RuntimeType = kubetypes.DockerContainerRuntime
	return joinOptions
}

func join(opt *common.JoinOptions, step *common.Step) error {
	step.Printf("Create the necessary directories")
	if err := createDirs(); err != nil {
		return err
	}

	// Do not create any files in the management directory,
	// you need to mount the contents of the mirror first.
	if err := request(opt, step); err != nil {
		return err
	}

	step.Printf("Generate systemd service file")
	if err := common.GenerateServiceFile(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName); err != nil {
		return fmt.Errorf("create systemd service file failed: %v", err)
	}

	step.Printf("Generate EdgeCore default configuration")
	if err := createEdgeConfigFiles(opt); err != nil {
		return fmt.Errorf("create edge config file failed: %v", err)
	}

	step.Printf("Run EdgeCore daemon")
	return runEdgeCore()
}

func createDirs() error {
	// Create management directory
	if err := os.MkdirAll(util.KubeEdgePath, os.ModePerm); err != nil {
		return fmt.Errorf("create %s folder path failed: %v", util.KubeEdgePath, err)
	}
	// Create config directory
	if err := os.MkdirAll(util.KubeEdgeConfigDir, os.ModePerm); err != nil {
		return fmt.Errorf("create %s folder path failed: %v", util.KubeEdgeConfigDir, err)
	}
	// Create log directory
	if err := os.MkdirAll(util.KubeEdgeLogPath, os.ModePerm); err != nil {
		return fmt.Errorf("create %s folder path failed: %v", util.KubeEdgeLogPath, err)
	}
	// Create resource directory
	if err := os.MkdirAll(util.KubeEdgeSocketPath, os.ModePerm); err != nil {
		return fmt.Errorf("create %s folder path failed: %v", util.KubeEdgeSocketPath, err)
	}
	return nil
}

func createEdgeConfigFiles(opt *common.JoinOptions) error {
	var edgeCoreConfig *v1alpha1.EdgeCoreConfig

	configFilePath := filepath.Join(util.KubeEdgePath, "config/edgecore.yaml")
	_, err := os.Stat(configFilePath)
	if err == nil || os.IsExist(err) {
		klog.Infoln("Read existing configuration file")
		b, err := os.ReadFile(configFilePath)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(b, &edgeCoreConfig); err != nil {
			return err
		}
	}
	if edgeCoreConfig == nil {
		klog.Infoln("The configuration does not exist or the parsing fails, and the default configuration is generated")
		edgeCoreConfig = v1alpha1.NewDefaultEdgeCoreConfig()
	}

	edgeCoreConfig.Modules.EdgeHub.WebSocket.Server = opt.CloudCoreIPPort
	if opt.Token != "" {
		edgeCoreConfig.Modules.EdgeHub.Token = opt.Token
	}
	if opt.EdgeNodeName != "" {
		edgeCoreConfig.Modules.Edged.HostnameOverride = opt.EdgeNodeName
	}
	if opt.RuntimeType != "" {
		edgeCoreConfig.Modules.Edged.RuntimeType = opt.RuntimeType
	}

	switch opt.CGroupDriver {
	case v1alpha1.CGroupDriverSystemd:
		edgeCoreConfig.Modules.Edged.CGroupDriver = v1alpha1.CGroupDriverSystemd
	case v1alpha1.CGroupDriverCGroupFS:
		edgeCoreConfig.Modules.Edged.CGroupDriver = v1alpha1.CGroupDriverCGroupFS
	default:
		return fmt.Errorf("unsupported CGroupDriver: %s", opt.CGroupDriver)
	}
	edgeCoreConfig.Modules.Edged.CGroupDriver = opt.CGroupDriver

	if opt.RemoteRuntimeEndpoint != "" {
		edgeCoreConfig.Modules.Edged.RemoteRuntimeEndpoint = opt.RemoteRuntimeEndpoint
		edgeCoreConfig.Modules.Edged.RemoteImageEndpoint = opt.RemoteRuntimeEndpoint
	}

	host, _, err := net.SplitHostPort(opt.CloudCoreIPPort)
	if err != nil {
		return fmt.Errorf("get current host and port failed: %v", err)
	}
	if opt.CertPort != "" {
		edgeCoreConfig.Modules.EdgeHub.HTTPServer = "https://" + net.JoinHostPort(host, opt.CertPort)
	} else {
		edgeCoreConfig.Modules.EdgeHub.HTTPServer = "https://" + net.JoinHostPort(host, "10002")
	}
	edgeCoreConfig.Modules.EdgeStream.TunnelServer = net.JoinHostPort(host, strconv.Itoa(constants.DefaultTunnelPort))

	if len(opt.Labels) > 0 {
		labelsMap := make(map[string]string)
		for _, label := range opt.Labels {
			arr := strings.SplitN(label, "=", 2)
			if arr[0] == "" {
				continue
			}

			if len(arr) > 1 {
				labelsMap[arr[0]] = arr[1]
			} else {
				labelsMap[arr[0]] = ""
			}
		}
		edgeCoreConfig.Modules.Edged.Labels = labelsMap
	}

	if errs := validation.ValidateEdgeCoreConfiguration(edgeCoreConfig); len(errs) > 0 {
		return errors.New(pkgutil.SpliceErrors(errs.ToAggregate().Errors()))
	}
	return common.Write2File(configFilePath, edgeCoreConfig)
}

func runEdgeCore() error {
	systemdExist := util.HasSystemd()

	var binExec, tip string
	if systemdExist {
		tip = fmt.Sprintf("KubeEdge edgecore is running, For logs visit: journalctl -u %s.service -xe", common.EdgeCore)
		binExec = fmt.Sprintf(
			"sudo systemctl daemon-reload && sudo systemctl enable %s && sudo systemctl start %s",
			common.EdgeCore, common.EdgeCore)
	} else {
		tip = fmt.Sprintf("KubeEdge edgecore is running, For logs visit: %s%s.log", util.KubeEdgeLogPath, util.KubeEdgeBinaryName)
		binExec = fmt.Sprintf(
			"%s > %skubeedge/edge/%s.log 2>&1 &",
			filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName),
			util.KubeEdgePath,
			util.KubeEdgeBinaryName,
		)
	}

	cmd := util.NewCommand(binExec)
	if err := cmd.Exec(); err != nil {
		return err
	}
	klog.Infoln(cmd.GetStdOut())
	klog.Infoln(tip)
	return nil
}
