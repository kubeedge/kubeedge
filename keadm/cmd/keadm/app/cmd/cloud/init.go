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

package cloud

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/helm"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	cloudInitLongDescription = `
"keadm init" command install KubeEdge's master node (on the cloud) component by using a list of set flags like helm.
It checks if the Kubernetes Master are installed already,
If not installed, please install the Kubernetes first.
`
	cloudInitExample = `
keadm init
- This command will render and install the Charts for Kubeedge cloud component

keadm init --advertise-address=127.0.0.1 --profile version=v%s --kube-config=/root/.kube/config
  - kube-config is the absolute path of kubeconfig which used to secure connectivity between cloudcore and kube-apiserver
	- a list of helm style set flags like "--set key=value" can be implemented, ref: https://github.com/kubeedge/kubeedge/tree/master/manifests/charts/cloudcore/README.md
`
)

// NewCloudInit represents the keadm init command for cloud component
func NewCloudInit() *cobra.Command {
	opts := newInitOptions()
	var cmd = &cobra.Command{
		Use:     "init",
		Short:   "Bootstraps cloud component. Checks and install (if required) the pre-requisites.",
		Long:    cloudInitLongDescription,
		Example: fmt.Sprintf(cloudInitExample, types.DefaultKubeEdgeVersion),
		RunE: func(cmd *cobra.Command, args []string) error {
			tool := helm.NewCloudCoreHelmTool(opts.KubeConfig, opts.KubeEdgeVersion)
			return tool.Install(opts)
		},
	}

	addInitOtherFlags(cmd, opts)
	addHelmValueOptionsFlags(cmd, opts)
	addForceOptionsFlags(cmd, opts)
	return cmd
}

// newInitOptions will initialise new instance of options everytime
func newInitOptions() *types.InitOptions {
	opts := &types.InitOptions{}
	opts.KubeConfig = types.DefaultKubeConfig

	return opts
}

func addInitOtherFlags(cmd *cobra.Command, initOpts *types.InitOptions) {
	cmd.Flags().StringVar(&initOpts.KubeEdgeVersion, types.FlagNameKubeEdgeVersion, initOpts.KubeEdgeVersion,
		"Use this key to set the default image tag")

	cmd.Flags().StringVar(&initOpts.AdvertiseAddress, types.FlagNameAdvertiseAddress, initOpts.AdvertiseAddress,
		"Use this key to set IPs in cloudcore's certificate SubAltNames field. eg: 10.10.102.78,10.10.102.79")

	cmd.Flags().StringVar(&initOpts.KubeConfig, types.FlagNameKubeConfig, initOpts.KubeConfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")

	cmd.Flags().StringVar(&initOpts.Manifests, types.FlagNameManifests, initOpts.Manifests,
		"Allow appending file directories of k8s resources to keadm, separated by commas")

	cmd.Flags().StringVarP(&initOpts.Manifests, types.FlagNameFiles, "f", initOpts.Manifests,
		"Allow appending file directories of k8s resources to keadm, separated by commas")

	cmd.Flags().BoolVarP(&initOpts.DryRun, types.FlagNameDryRun, "d", initOpts.DryRun,
		"Print the generated k8s resources on the stdout, not actual execute. Always use in debug mode")

	cmd.Flags().StringVar(&initOpts.ExternalHelmRoot, types.FlagNameExternalHelmRoot, initOpts.ExternalHelmRoot,
		"Add external helm root path to keadm.")

	cmd.Flags().StringVar(&initOpts.ImageRepository, types.FlagNameImageRepository, initOpts.ImageRepository,
		"Choose a container image repository to pull the image of the kubedge component.")
}

func addHelmValueOptionsFlags(cmd *cobra.Command, initOpts *types.InitOptions) {
	cmd.Flags().StringArrayVar(&initOpts.Sets, types.FlagNameSet, []string{}, "Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	cmd.Flags().StringVar(&initOpts.Profile, types.FlagNameProfile, initOpts.Profile, fmt.Sprintf("Set profile on the command line (/path/version.yaml or version=v%s)", types.DefaultKubeEdgeVersion))
}

func addForceOptionsFlags(cmd *cobra.Command, initOpts *types.InitOptions) {
	cmd.Flags().BoolVar(&initOpts.Force, types.FlagNameForce, initOpts.Force,
		"Forced installing the cloud components without waiting.")
}

// AddInit2ToolsList reads the flagData (containing val and default val) and join options to fill the list of tools.
func AddInit2ToolsList(toolList map[string]types.ToolsInstaller, initOpts *types.InitOptions) error {
	common := util.Common{
		ToolVersion: semver.MustParse(util.GetHelmVersion(initOpts.KubeEdgeVersion, util.RetryTimes)),
		KubeConfig:  initOpts.KubeConfig,
	}
	toolList["helm"] = &helm.KubeCloudHelmInstTool{
		Common:           common,
		AdvertiseAddress: initOpts.AdvertiseAddress,
		Manifests:        initOpts.Manifests,
		Namespace:        constants.SystemNamespace,
		DryRun:           initOpts.DryRun,
		Sets:             initOpts.Sets,
		Profile:          initOpts.Profile,
		ExternalHelmRoot: initOpts.ExternalHelmRoot,
		Force:            initOpts.Force,
		Action:           types.HelmInstallAction,
	}
	return nil
}

// ExecuteInit the installation for each tool and start cloudcore
func ExecuteInit(toolList map[string]types.ToolsInstaller) error {
	return toolList["helm"].InstallTools()
}
