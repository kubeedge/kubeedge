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
	helm "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/helm"
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

keadm beta init --advertise-address=127.0.0.1 --profile version=v1.9.0 --kube-config=/root/.kube/config

  - kube-config is the absolute path of kubeconfig which used to secure connectivity between cloudcore and kube-apiserver
`
)

// NewInitBeta represents the beta version of keadm init command for cloud component
func NewInitBeta() *cobra.Command {
	BetaInit := newInitBetaOptions()

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
			err := AddInitBeta2ToolsList(tools, flagVals, BetaInit)
			if err != nil {
				return err
			}
			return ExecuteInitBeta(tools)
		},
	}

	addInitBetaJoinOtherFlags(cmd, BetaInit)
	addHelmValueOptionsFlags(cmd, BetaInit)
	addForceOptionsFlags(cmd, BetaInit)
	return cmd
}

//newInitBetaOptions will initialise new instance of options everytime
func newInitBetaOptions() *types.InitBetaOptions {
	opts := &types.InitBetaOptions{}
	opts.KubeConfig = types.DefaultKubeConfig

	return opts
}

func addInitBetaJoinOtherFlags(cmd *cobra.Command, initBetaOpts *types.InitBetaOptions) {
	cmd.Flags().StringVar(&initBetaOpts.KubeEdgeVersion, types.KubeEdgeVersion, initBetaOpts.KubeEdgeVersion,
		"Use this key to set the default image tag")

	cmd.Flags().StringVar(&initBetaOpts.AdvertiseAddress, types.AdvertiseAddress, initBetaOpts.AdvertiseAddress,
		"Use this key to set IPs in cloudcore's certificate SubAltNames field. eg: 10.10.102.78,10.10.102.79")

	cmd.Flags().StringVar(&initBetaOpts.KubeConfig, types.KubeConfig, initBetaOpts.KubeConfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")

	cmd.Flags().StringVar(&initBetaOpts.Manifests, types.Manifests, initBetaOpts.Manifests,
		"Allow appending file directories of k8s resources to keadm, separated by commas")

	cmd.Flags().StringVarP(&initBetaOpts.Manifests, types.Files, "f", initBetaOpts.Manifests,
		"Allow appending file directories of k8s resources to keadm, separated by commas")

	cmd.Flags().BoolVarP(&initBetaOpts.DryRun, types.DryRun, "d", initBetaOpts.DryRun,
		"Print the generated k8s resources on the stdout, not actual excute. Always use in debug mode")

	cmd.Flags().StringVar(&initBetaOpts.ExternalHelmRoot, types.ExternalHelmRoot, initBetaOpts.ExternalHelmRoot,
		"Add external helm root path to keadm.")
}

func addHelmValueOptionsFlags(cmd *cobra.Command, initBetaOpts *types.InitBetaOptions) {
	cmd.Flags().StringArrayVar(&initBetaOpts.Sets, "set", []string{}, "Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	cmd.Flags().StringVar(&initBetaOpts.Profile, "profile", initBetaOpts.Profile, "Set profile on the command line (iptablesMgrMode=external or version=v1.9.1)")
}

func addForceOptionsFlags(cmd *cobra.Command, initBetaOpts *types.InitBetaOptions) {
	cmd.Flags().BoolVar(&initBetaOpts.Force, types.Force, initBetaOpts.Force,
		"Forced installing the cloud components without waiting.")
}

//AddInitBeta2ToolsList reads the flagData (containing val and default val) and join options to fill the list of tools.
func AddInitBeta2ToolsList(toolList map[string]types.ToolsInstaller, flagData map[string]types.FlagData, initBetaOpts *types.InitBetaOptions) error {
	var latestVersion string
	var kubeedgeVersion string
	for i := 0; i < util.RetryTimes; i++ {
		version, err := util.GetLatestVersion()
		if err != nil {
			fmt.Println("Failed to get the latest KubeEdge release version, error: ", err)
			continue
		}
		if len(version) > 0 {
			kubeedgeVersion = strings.TrimPrefix(version, "v")
			latestVersion = version
			break
		}
	}
	if len(latestVersion) == 0 {
		kubeedgeVersion = types.DefaultKubeEdgeVersion
		fmt.Println("Failed to get the latest KubeEdge release version, will use default version: ", kubeedgeVersion)
	}

	common := util.Common{
		ToolVersion: semver.MustParse(kubeedgeVersion),
		KubeConfig:  initBetaOpts.KubeConfig,
	}
	toolList["helm"] = &helm.KubeCloudHelmInstTool{
		Common:           common,
		AdvertiseAddress: initBetaOpts.AdvertiseAddress,
		KubeEdgeVersion:  initBetaOpts.KubeEdgeVersion,
		Manifests:        initBetaOpts.Manifests,
		Namespace:        constants.SystemNamespace,
		DryRun:           initBetaOpts.DryRun,
		Sets:             initBetaOpts.Sets,
		Profile:          initBetaOpts.Profile,
		ExternalHelmRoot: initBetaOpts.ExternalHelmRoot,
		Force:            initBetaOpts.Force,
		Action:           types.HelmInstallAction,
	}
	return nil
}

//ExecuteInitBeta the installation for each tool and start cloudcore
func ExecuteInitBeta(toolList map[string]types.ToolsInstaller) error {
	return toolList["helm"].InstallTools()
}
