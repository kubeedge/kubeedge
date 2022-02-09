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

package beta

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	cloudBetaInitLongDescription = `
"keadm beta init" command install KubeEdge's master node (on the cloud) component by using a list of set flags like helm.
It checks if the Kubernetes Master are installed already,
If not installed, please install the Kubernetes first.
`
	cloudBetaInitExample = `
keadm beta init

- This command will render and install the Charts for Kubeedge cloud component

keadm beta init --advertise-address=127.0.0.1 [--set cloudcore-tag=v1.9.0] --profile version=v1.9.0 -n kubeedge --kube-config=/root/.kube/config

  - kube-config is the absolute path of kubeconfig which used to secure connectivity between cloudcore and kube-apiserver
`
)

// NewBetaInit represents the beta version of keadm init command for cloud component
func NewBetaInit() *cobra.Command {
	BetaInit := newBetaInitOptions()

	tools := make(map[string]types.ToolsInstaller)
	flagVals := make(map[string]types.FlagData)

	var cmd = &cobra.Command{
		Use:     "init",
		Short:   "Bootstraps cloud component. Checks and install (if required) the pre-requisites.",
		Long:    cloudBetaInitLongDescription,
		Example: cloudBetaInitExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFlags := func(f *pflag.Flag) {
				util.AddToolVals(f, flagVals)
			}
			cmd.Flags().VisitAll(checkFlags)
			err := AddBetaInit2ToolsList(tools, flagVals, BetaInit)
			if err != nil {
				return err
			}
			return ExecuteBetaInit(tools)
		},
	}

	addBetaInitJoinOtherFlags(cmd, BetaInit)
	addHelmValueOptionsFlags(cmd, BetaInit)
	addForceOptionsFlags(cmd, BetaInit)
	return cmd
}

//newBetaInitOptions will initialise new instance of options everytime
func newBetaInitOptions() *types.BetaInitOptions {
	opts := &types.BetaInitOptions{}
	opts.KubeConfig = types.DefaultKubeConfig
	return opts
}

func addBetaInitJoinOtherFlags(cmd *cobra.Command, BetaInitOpts *types.BetaInitOptions) {
	cmd.Flags().StringVar(&BetaInitOpts.AdvertiseAddress, types.AdvertiseAddress, BetaInitOpts.AdvertiseAddress,
		"Use this key to set IPs in cloudcore's certificate SubAltNames field. eg: 10.10.102.78,10.10.102.79")

	cmd.Flags().StringVar(&BetaInitOpts.KubeConfig, types.KubeConfig, BetaInitOpts.KubeConfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")

	cmd.Flags().StringVar(&BetaInitOpts.Manifests, types.Manifests, BetaInitOpts.Manifests,
		"Allow appending file directories of k8s resources to keadm, separated by commas")

	cmd.Flags().StringVarP(&BetaInitOpts.Manifests, types.Files, "f", BetaInitOpts.Manifests,
		"Allow appending file directories of k8s resources to keadm, separated by commas")

	cmd.Flags().BoolVarP(&BetaInitOpts.DryRun, types.DryRun, "d", BetaInitOpts.DryRun,
		"Print the generated k8s resources on the stdout, not actual excute. Always use in debug mode")

	cmd.Flags().StringVar(&BetaInitOpts.CloudcoreImage, types.CloudcoreImage, BetaInitOpts.CloudcoreImage,
		"The image repo of the cloudcore, default is kubeedge/cloudcore")

	cmd.Flags().StringVar(&BetaInitOpts.CloudcoreTag, types.CloudcoreTag, BetaInitOpts.CloudcoreTag,
		"The image tag of the cloudcore, default is v1.9.0")

	cmd.Flags().StringVar(&BetaInitOpts.IptablesMgrImage, types.IptablesMgrImage, BetaInitOpts.IptablesMgrImage,
		"The image repo of the iptables manager, default is kubeedge/iptables-manager")

	cmd.Flags().StringVar(&BetaInitOpts.IptablesMgrTag, types.IptablesMgrTag, BetaInitOpts.IptablesMgrTag,
		"The image tag of the iptables manager, default is v1.9.0")

	cmd.Flags().StringVar(&BetaInitOpts.ExternalHelmRoot, types.ExternalHelmRoot, BetaInitOpts.ExternalHelmRoot,
		"Add external helm root path to keadm.")
}

func addHelmValueOptionsFlags(cmd *cobra.Command, BetaInitOpts *types.BetaInitOptions) {
	cmd.Flags().StringArrayVar(&BetaInitOpts.Sets, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	cmd.Flags().StringVar(&BetaInitOpts.Profile, "profile", BetaInitOpts.Profile, "set profile on the command line (iptablesMgrMode=external or version=1.9.1)")
}

func addForceOptionsFlags(cmd *cobra.Command, BetaInitOpts *types.BetaInitOptions) {
	cmd.Flags().BoolVar(&BetaInitOpts.Force, types.Force, BetaInitOpts.Force,
		"Forced installing the cloud components.")
}

//AddBetaInit2ToolsList reads the flagData (containing val and default val) and join options to fill the list of tools.
func AddBetaInit2ToolsList(toolList map[string]types.ToolsInstaller, flagData map[string]types.FlagData, BetaInitOptions *types.BetaInitOptions) error {
	var kubeVer string
	var latestVersion string
	for i := 0; i < util.RetryTimes; i++ {
		version, err := util.GetLatestVersion()
		if err != nil {
			fmt.Println("Failed to get the latest KubeEdge release version, error: ", err)
			continue
		}
		if len(version) > 0 {
			kubeVer = strings.TrimPrefix(version, "v")
			latestVersion = version
			break
		}
	}
	if len(latestVersion) == 0 {
		kubeVer = types.DefaultKubeEdgeVersion
		fmt.Println("Failed to get the latest KubeEdge release version, will use default version: ", kubeVer)
	}

	if BetaInitOptions.Namespace == "" {
		BetaInitOptions.Namespace = constants.SystemNamespace
	}

	common := util.Common{
		ToolVersion: semver.MustParse(kubeVer),
		KubeConfig:  BetaInitOptions.KubeConfig,
	}
	toolList["helm"] = &util.KubeCloudHelmInstTool{
		Common:           common,
		AdvertiseAddress: BetaInitOptions.AdvertiseAddress,
		Manifests:        BetaInitOptions.Manifests,
		Namespace:        constants.SystemNamespace,
		DryRun:           BetaInitOptions.DryRun,
		CloudcoreImage:   BetaInitOptions.CloudcoreImage,
		CloudcoreTag:     BetaInitOptions.CloudcoreTag,
		IptablesMgrImage: BetaInitOptions.IptablesMgrImage,
		IptablesMgrTag:   BetaInitOptions.IptablesMgrTag,
		Sets:             BetaInitOptions.Sets,
		Profile:          BetaInitOptions.Profile,
		ExternalHelmRoot: BetaInitOptions.ExternalHelmRoot,
		Force:            BetaInitOptions.Force,
		Action:           types.HelmInstallAction,
	}
	return nil
}

//ExecuteBetaInit the installation for each tool and start cloudcore
func ExecuteBetaInit(toolList map[string]types.ToolsInstaller) error {
	return toolList["helm"].InstallTools()
}
