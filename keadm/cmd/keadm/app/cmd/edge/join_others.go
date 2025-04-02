//go:build !windows

/*
Copyright 2023 The KubeEdge Authors.

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
	"time"

	"github.com/opencontainers/selinux/go-selinux"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2/validation"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/extsystem"
	pkgutil "github.com/kubeedge/kubeedge/pkg/util"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

func AddJoinOtherFlags(cmd *cobra.Command, joinOptions *common.JoinOptions) {
	cmd.Flags().StringVar(&joinOptions.KubeEdgeVersion, common.FlagNameKubeEdgeVersion, joinOptions.KubeEdgeVersion,
		"Use this key to download and use the required KubeEdge version")
	cmd.Flags().Lookup(common.FlagNameKubeEdgeVersion).NoOptDefVal = joinOptions.KubeEdgeVersion

	cmd.Flags().StringVar(&joinOptions.CGroupDriver, common.FlagNameCGroupDriver, joinOptions.CGroupDriver,
		"CGroupDriver that uses to manipulate cgroups on the host (cgroupfs or systemd), the default value is cgroupfs")

	cmd.Flags().StringVar(&joinOptions.CertPath, common.FlagNameCertPath, joinOptions.CertPath,
		fmt.Sprintf("The certPath used by edgecore, the default value is %s", common.DefaultCertPath))

	cmd.Flags().StringVarP(&joinOptions.CloudCoreIPPort, common.FlagNameCloudCoreIPPort, "e", joinOptions.CloudCoreIPPort,
		"IP:Port address of KubeEdge CloudCore")

	if err := cmd.MarkFlagRequired(common.FlagNameCloudCoreIPPort); err != nil {
		fmt.Printf("mark flag required failed with error: %v\n", err)
	}

	cmd.Flags().StringVarP(&joinOptions.EdgeNodeName, common.FlagNameEdgeNodeName, "i", joinOptions.EdgeNodeName,
		"KubeEdge Node unique identification string, if flag not used then the command will generate a unique id on its own")

	cmd.Flags().StringVarP(&joinOptions.RemoteRuntimeEndpoint, common.FlagNameRemoteRuntimeEndpoint, "p", joinOptions.RemoteRuntimeEndpoint,
		"KubeEdge Edge Node RemoteRuntimeEndpoint string.")

	cmd.Flags().StringVarP(&joinOptions.Token, common.FlagNameToken, "t", joinOptions.Token,
		"Used for edge to apply for the certificate")

	cmd.Flags().StringVarP(&joinOptions.CertPort, common.FlagNameCertPort, "s", joinOptions.CertPort,
		"The port where to apply for the edge certificate")

	cmd.Flags().StringVar(&joinOptions.TarballPath, common.FlagNameTarballPath, joinOptions.TarballPath,
		"Use this key to set the temp directory path for KubeEdge tarball, if not exist, download it")

	cmd.Flags().StringSliceVarP(&joinOptions.Labels, common.FlagNameLabels, "l", joinOptions.Labels,
		`Use this key to set the customized labels for node, you can input customized labels like key1=value1,key2=value2`)

	// FIXME: cleanup this code when the static pod mqtt broker no longer needs to be compatible
	cmd.Flags().BoolVar(&joinOptions.WithMQTT, "with-mqtt", joinOptions.WithMQTT,
		`Use this key to set whether to install and start MQTT Broker by default`)
	if err := cmd.Flags().MarkDeprecated("with-mqtt",
		"The mqtt broker is alreay managed by the DaemonSet in the cloud"); err != nil {
		klog.Warningf("failed to mark the flag with-mqtt to deprecated, err: %v", err)
	}

	cmd.Flags().StringVar(&joinOptions.ImageRepository, common.FlagNameImageRepository, joinOptions.ImageRepository,
		`Use this key to decide which image repository to pull images from`,
	)

	cmd.Flags().StringVar(&joinOptions.HubProtocol, common.HubProtocol, joinOptions.HubProtocol,
		`Use this key to decide which communication protocol the edge node adopts.`)

	cmd.Flags().StringArrayVar(&joinOptions.Sets, common.FlagNameSet, joinOptions.Sets,
		`Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)`)

	cmd.Flags().StringVar(&joinOptions.PreRun, common.FlagNamePreRun, joinOptions.PreRun,
		`Execute the prescript before joining the node. (for example: keadm join --pre-run=./test-script.sh ...)`)

	cmd.Flags().StringVar(&joinOptions.PostRun, common.FlagNamePostRun, joinOptions.PostRun,
		`Execute the postscript after joining the node. (for example: keadm join --post-run=./test-script.sh ...)`)
}

func createEdgeConfigFiles(opt *common.JoinOptions) error {
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
		edgeCoreConfig = v1alpha2.NewDefaultEdgeCoreConfig()
	}

	// TODO: remove this after release 1.14
	// this is for keeping backward compatibility
	// don't save token in configuration edgecore.yaml
	if opt.Token != "" {
		edgeCoreConfig.Modules.EdgeHub.Token = opt.Token
	}
	if opt.EdgeNodeName != "" {
		edgeCoreConfig.Modules.Edged.HostnameOverride = opt.EdgeNodeName
	}

	switch opt.CGroupDriver {
	case v1alpha2.CGroupDriverSystemd:
		edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.CgroupDriver = v1alpha2.CGroupDriverSystemd
	case v1alpha2.CGroupDriverCGroupFS:
		edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.CgroupDriver = v1alpha2.CGroupDriverCGroupFS
	default:
		return fmt.Errorf("unsupported CGroupDriver: %s", opt.CGroupDriver)
	}

	if opt.RemoteRuntimeEndpoint != "" {
		edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint = opt.RemoteRuntimeEndpoint
		edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.ImageServiceEndpoint = opt.RemoteRuntimeEndpoint
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

	switch opt.HubProtocol {
	case api.ProtocolTypeQuic:
		edgeCoreConfig.Modules.EdgeHub.Quic.Enable = true
		edgeCoreConfig.Modules.EdgeHub.WebSocket.Enable = false
		edgeCoreConfig.Modules.EdgeHub.Quic.Server = opt.CloudCoreIPPort
		edgeCoreConfig.Modules.EdgeHub.WebSocket.Server = net.JoinHostPort(host, strconv.Itoa(constants.DefaultWebSocketPort))
	case api.ProtocolTypeWS:
		edgeCoreConfig.Modules.EdgeHub.Quic.Enable = false
		edgeCoreConfig.Modules.EdgeHub.WebSocket.Enable = true
		edgeCoreConfig.Modules.EdgeHub.Quic.Server = net.JoinHostPort(host, strconv.Itoa(constants.DefaultQuicPort))
		edgeCoreConfig.Modules.EdgeHub.WebSocket.Server = opt.CloudCoreIPPort
	default:
		return fmt.Errorf("unsupported hub of protocol: %s", opt.HubProtocol)
	}
	edgeCoreConfig.Modules.EdgeStream.TunnelServer = net.JoinHostPort(host, strconv.Itoa(constants.DefaultTunnelPort))

	if len(opt.Labels) > 0 {
		edgeCoreConfig.Modules.Edged.NodeLabels = setEdgedNodeLabels(opt)
	}
	if len(opt.Sets) > 0 {
		for _, set := range opt.Sets {
			if err := util.ParseSet(edgeCoreConfig, set); err != nil {
				return err
			}
		}
	}

	if errs := validation.ValidateEdgeCoreConfiguration(edgeCoreConfig); len(errs) > 0 {
		return errors.New(pkgutil.SpliceErrors(errs.ToAggregate().Errors()))
	}
	return common.Write2File(configFilePath, edgeCoreConfig)
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
	extSystem, err := extsystem.GetExtSystem()
	if err != nil {
		return fmt.Errorf("failed to get ext system, err: %v", err)
	}
	if err := extSystem.ServiceCreate(util.KubeEdgeBinaryName,
		filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName),
		map[string]string{
			constants.DeployMqttContainerEnv: strconv.FormatBool(opt.WithMQTT),
		},
	); err != nil {
		return fmt.Errorf("failed to create edgecore systemd service, err: %v", err)
	}

	// write token to bootstrap configure file
	if err := createBootstrapFile(opt); err != nil {
		return fmt.Errorf("create bootstrap file failed: %v", err)
	}
	// Delete the bootstrap file, so the credential used for TLS bootstrap is removed from disk
	defer os.Remove(constants.BootstrapFile)

	step.Printf("Generate EdgeCore default configuration")
	if err := createEdgeConfigFiles(opt); err != nil {
		return fmt.Errorf("create edge config file failed: %v", err)
	}

	step.Printf("Run EdgeCore daemon")
	if err := runEdgeCore(opt.WithMQTT); err != nil {
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

	if err != nil {
		return err
	}
	step.Printf("Install Complete!")
	return err
}

func runEdgeCore(withMqtt bool) error {
	//If selinux is enabled, it is necessary to modify the context of edgecore, as edgecore is copied from the container
	//and will be marked as container_file_t by selinux in container, the marked file cannot be operated by the host process.
	err := selinuxLabelRevision(selinux.GetEnabled())
	if err != nil {
		return err
	}
	systemdExist := util.HasSystemd()

	var binExec, tip string
	if systemdExist {
		tip = fmt.Sprintf("KubeEdge edgecore is running, For logs visit: journalctl -u %s.service -xe", common.EdgeCore)
		binExec = fmt.Sprintf(
			"sudo systemctl daemon-reload && sudo systemctl enable %s && sudo systemctl start %s",
			common.EdgeCore, common.EdgeCore)
	} else {
		tip = fmt.Sprintf("KubeEdge edgecore is running, For logs visit: %s%s.log", util.KubeEdgeLogPath, util.KubeEdgeBinaryName)

		// FIXME: cleanup this code when the static pod mqtt broker no longer needs to be compatible
		err := os.Setenv(constants.DeployMqttContainerEnv, strconv.FormatBool(withMqtt))
		if err != nil {
			klog.Errorf("Set Environment %s failed, err: %v", constants.DeployMqttContainerEnv, err)
		}

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

func selinuxLabelRevision(enable bool) error {
	if !enable {
		return nil
	}

	label, err := selinux.FileLabel(filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName))
	if err != nil {
		return fmt.Errorf("get selinux context of edgecore faild with error:%w", err)
	}

	if label != util.EdgeCoreSELinuxLabel {
		if err = selinux.SetFileLabel(filepath.Join(util.KubeEdgeUsrBinPath, util.KubeEdgeBinaryName), util.EdgeCoreSELinuxLabel); err != nil {
			return fmt.Errorf("reset selinux context on edgecore faild with error:%w", err)
		}
	}
	return nil
}
