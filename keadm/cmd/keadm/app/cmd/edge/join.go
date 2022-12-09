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

package edge

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	validationv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1/validation"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2/validation"
	pkgutil "github.com/kubeedge/kubeedge/pkg/util"
)

var (
	edgeJoinDescription = `
"keadm join" command bootstraps KubeEdge's worker node (at the edge) component.
It will also connect with cloud component to receive
further instructions and forward telemetry data from
devices to cloud
`
	edgeJoinExample = `
keadm join --cloudcore-ipport=<ip:port address> --edgenode-name=<unique string as edge identifier>

  - For this command --cloudcore-ipport flag is a required option
  - This command will download and install the default version of pre-requisites and KubeEdge

keadm join --cloudcore-ipport=10.20.30.40:10000 --edgenode-name=testing123 --kubeedge-version=v` + common.DefaultKubeEdgeVersion + `
`
)

var edgeCoreConfig *v1alpha2.EdgeCoreConfig

func NewEdgeJoin() *cobra.Command {
	joinOptions := newOption()
	step := common.NewStep()
	cmd := &cobra.Command{
		Use:          "join",
		Short:        "Bootstraps edge component. Checks and install (if required) the pre-requisites. Execute it on any edge node machine you wish to join",
		Long:         edgeJoinDescription,
		Example:      edgeJoinExample,
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			step.Printf("Check KubeEdge edgecore process status")
			running, err := util.IsKubeEdgeProcessRunning(util.KubeEdgeBinaryName)
			if err != nil {
				return fmt.Errorf("check KubeEdge edgecore process status failed: %v", err)
			}
			if running {
				return fmt.Errorf("EdgeCore is already running on this node, please run reset to clean up first")
			}

			step.Printf("Check if the management directory is clean")
			if _, err := os.Stat(util.KubeEdgePath); err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return fmt.Errorf("Stat management directory %s failed: %v", util.KubeEdgePath, err)
			}
			entries, err := os.ReadDir(util.KubeEdgePath)
			if err != nil {
				return fmt.Errorf("read management directory %s failed: %v", util.KubeEdgePath, err)
			}
			if len(entries) > 0 {
				return fmt.Errorf("the management directory %s is not clean, please remove it first", util.KubeEdgePath)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ver, err := util.GetCurrentVersion(joinOptions.KubeEdgeVersion)
			if err != nil {
				return fmt.Errorf("edge node join failed: %v", err)
			}
			joinOptions.KubeEdgeVersion = ver

			if err := join(joinOptions, step); err != nil {
				return fmt.Errorf("edge node join failed: %v", err)
			}

			return nil
		},
	}

	AddJoinOtherFlags(cmd, joinOptions)
	return cmd
}

func AddJoinOtherFlags(cmd *cobra.Command, joinOptions *common.JoinOptions) {
	cmd.Flags().StringVar(&joinOptions.KubeEdgeVersion, common.KubeEdgeVersion, joinOptions.KubeEdgeVersion,
		"Use this key to download and use the required KubeEdge version")
	cmd.Flags().Lookup(common.KubeEdgeVersion).NoOptDefVal = joinOptions.KubeEdgeVersion

	cmd.Flags().StringVar(&joinOptions.CGroupDriver, common.CGroupDriver, joinOptions.CGroupDriver,
		"CGroupDriver that uses to manipulate cgroups on the host (cgroupfs or systemd), the default value is cgroupfs")

	cmd.Flags().StringVar(&joinOptions.CertPath, common.CertPath, joinOptions.CertPath,
		fmt.Sprintf("The certPath used by edgecore, the default value is %s", common.DefaultCertPath))

	cmd.Flags().StringVarP(&joinOptions.CloudCoreIPPort, common.CloudCoreIPPort, "e", joinOptions.CloudCoreIPPort,
		"IP:Port address of KubeEdge CloudCore")

	if err := cmd.MarkFlagRequired(common.CloudCoreIPPort); err != nil {
		fmt.Printf("mark flag required failed with error: %v\n", err)
	}

	cmd.Flags().StringVarP(&joinOptions.RuntimeType, common.RuntimeType, "r", joinOptions.RuntimeType,
		"Container runtime type")

	cmd.Flags().StringVarP(&joinOptions.EdgeNodeName, common.EdgeNodeName, "i", joinOptions.EdgeNodeName,
		"KubeEdge Node unique identification string, If flag not used then the command will generate a unique id on its own")

	cmd.Flags().StringVarP(&joinOptions.RemoteRuntimeEndpoint, common.RemoteRuntimeEndpoint, "p", joinOptions.RemoteRuntimeEndpoint,
		"KubeEdge Edge Node RemoteRuntimeEndpoint string, If flag not set, it will use unix:///run/containerd/containerd.sock")

	cmd.Flags().StringVarP(&joinOptions.Token, common.Token, "t", joinOptions.Token,
		"Used for edge to apply for the certificate")

	cmd.Flags().StringVarP(&joinOptions.CertPort, common.CertPort, "s", joinOptions.CertPort,
		"The port where to apply for the edge certificate")

	cmd.Flags().StringVar(&joinOptions.TarballPath, common.TarballPath, joinOptions.TarballPath,
		"Use this key to set the temp directory path for KubeEdge tarball, if not exist, download it")

	cmd.Flags().StringSliceVarP(&joinOptions.Labels, common.Labels, "l", joinOptions.Labels,
		`use this key to set the customized labels for node. you can input customized labels like key1=value1,key2=value2`)

	cmd.Flags().BoolVar(&joinOptions.WithMQTT, "with-mqtt", joinOptions.WithMQTT,
		`use this key to set whether to install and start MQTT Broker by default`)

	cmd.Flags().StringVar(&joinOptions.ImageRepository, common.ImageRepository, joinOptions.ImageRepository,
		`Use this key to decide which image repository to pull images from.`,
	)
}

func newOption() *common.JoinOptions {
	joinOptions := &common.JoinOptions{}
	joinOptions.WithMQTT = true
	joinOptions.CGroupDriver = v1alpha2.CGroupDriverCGroupFS
	joinOptions.CertPath = common.DefaultCertPath
	joinOptions.RuntimeType = kubetypes.RemoteContainerRuntime
	joinOptions.RemoteRuntimeEndpoint = constants.DefaultRemoteRuntimeEndpoint
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
	if err := common.GenerateServiceFile(util.KubeEdgeBinaryName, filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName)); err != nil {
		return fmt.Errorf("create systemd service file failed: %v", err)
	}

	// write token to bootstrap configure file
	if err := createBootstrapFile(opt); err != nil {
		return fmt.Errorf("create bootstrap file failed: %v", err)
	}
	// Delete the bootstrap file, so the credential used for TLS bootstrap is removed from disk
	defer os.Remove(filepath.Join(util.KubeEdgePath, "bootstrap-edgecore.conf"))

	step.Printf("Generate EdgeCore default configuration")
	if err := createEdgeConfigFiles(opt); err != nil {
		return fmt.Errorf("create edge config file failed: %v", err)
	}

	step.Printf("Run EdgeCore daemon")
	err := runEdgeCore()
	if err != nil {
		return fmt.Errorf("start edgecore failed: %v", err)
	}

	// wait for edgecore start successfully using specified token
	// if edgecore start, it will get ca/certs from cloud
	// if ca/certs generated, we can remove bootstrap file
	err = wait.Poll(10*time.Second, 300*time.Second, func() (bool, error) {
		if util.FileExists(edgeCoreConfig.Modules.EdgeHub.TLSCAFile) &&
			util.FileExists(edgeCoreConfig.Modules.EdgeHub.TLSCertFile) &&
			util.FileExists(edgeCoreConfig.Modules.EdgeHub.TLSPrivateKeyFile) {
			return true, nil
		}
		return false, nil
	})

	return err
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
	// Determines whether the kubeEdgeVersion is earlier than v1.12.0
	// If so, we need to create edgeconfig with v1alpha1 version
	v, err := semver.ParseTolerant(opt.KubeEdgeVersion)
	if err != nil {
		return fmt.Errorf("parse kubeedge version failed, %v", err)
	}
	if v.Major <= 1 && v.Minor < 12 {
		return createV1alpha1EdgeConfigFiles(opt)
	}

	configFilePath := filepath.Join(util.KubeEdgePath, "config/edgecore.yaml")
	_, err = os.Stat(configFilePath)
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
		edgeCoreConfig = v1alpha2.NewDefaultEdgeCoreConfig()
	}

	edgeCoreConfig.Modules.EdgeHub.WebSocket.Server = opt.CloudCoreIPPort
	// TODO: remove this after release 1.14
	// this is for keeping backward compatibility
	// don't save token in configuration edgecore.yaml
	if opt.Token != "" {
		edgeCoreConfig.Modules.EdgeHub.Token = opt.Token
	}
	if opt.EdgeNodeName != "" {
		edgeCoreConfig.Modules.Edged.HostnameOverride = opt.EdgeNodeName
	}
	if opt.RuntimeType != "" {
		edgeCoreConfig.Modules.Edged.ContainerRuntime = opt.RuntimeType
	}

	switch opt.CGroupDriver {
	case v1alpha2.CGroupDriverSystemd:
		edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.CgroupDriver = v1alpha2.CGroupDriverSystemd
	case v1alpha2.CGroupDriverCGroupFS:
		edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.CgroupDriver = v1alpha2.CGroupDriverCGroupFS
	default:
		return fmt.Errorf("unsupported CGroupDriver: %s", opt.CGroupDriver)
	}
	edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.CgroupDriver = opt.CGroupDriver

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
		edgeCoreConfig.Modules.Edged.NodeLabels = setEdgedNodeLabels(opt)
	}

	if errs := validation.ValidateEdgeCoreConfiguration(edgeCoreConfig); len(errs) > 0 {
		return errors.New(pkgutil.SpliceErrors(errs.ToAggregate().Errors()))
	}
	return common.Write2File(configFilePath, edgeCoreConfig)
}

func createV1alpha1EdgeConfigFiles(opt *common.JoinOptions) error {
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
		edgeCoreConfig.Modules.Edged.Labels = setEdgedNodeLabels(opt)
	}

	if errs := validationv1alpha1.ValidateEdgeCoreConfiguration(edgeCoreConfig); len(errs) > 0 {
		return errors.New(pkgutil.SpliceErrors(errs.ToAggregate().Errors()))
	}
	return common.Write2File(configFilePath, edgeCoreConfig)
}

func setEdgedNodeLabels(opt *common.JoinOptions) map[string]string {
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
	return labelsMap
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

func createBootstrapFile(opt *common.JoinOptions) error {
	bootstrapFile := constants.BootstrapFile
	_, err := os.Create(bootstrapFile)
	if err != nil {
		return err
	}

	// write token to bootstrap-edgecore.conf file
	token := []byte(opt.Token)
	return os.WriteFile(bootstrapFile, token, 0640)
}
