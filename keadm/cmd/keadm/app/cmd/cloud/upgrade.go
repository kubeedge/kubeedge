/*
Copyright 2024 The KubeEdge Authors.

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
	"github.com/spf13/cobra"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/helm"
)

func NewCloudUpgrade() *cobra.Command {
	opts := newCloudUpgradeOptions()

	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "Upgrade the cloud components",
		Long: "Upgrade the cloud components to the desired version, " +
			"it uses helm to upgrade the installed release of cloudcore chart, which includes all the cloud components",
		RunE: func(cmd *cobra.Command, args []string) error {
			tool := helm.NewCloudCoreHelmTool(opts.KubeConfig, opts.KubeEdgeVersion)
			return tool.Upgrade(opts)
		},
	}

	addUpgradeOptionFlags(cmd, opts)
	return cmd
}

func newCloudUpgradeOptions() *types.CloudUpgradeOptions {
	opts := &types.CloudUpgradeOptions{}
	opts.KubeConfig = types.DefaultKubeConfig
	return opts
}

func addUpgradeOptionFlags(cmd *cobra.Command, opts *types.CloudUpgradeOptions) {
	fs := cmd.Flags()

	fs.StringVar(&opts.KubeEdgeVersion, types.FlagNameKubeEdgeVersion, opts.KubeEdgeVersion,
		"Use this key to set the upgrade image tag")

	fs.StringVar(&opts.AdvertiseAddress, types.FlagNameAdvertiseAddress, opts.AdvertiseAddress,
		"Please set the same value as when you installed it, this value is only used to generate the configuration"+
			" and does not regenerate the certificate. eg: 10.10.102.78,10.10.102.79")

	fs.StringVar(&opts.KubeConfig, types.FlagNameKubeConfig, opts.KubeConfig,
		"Use this key to update kube-config path, eg: $HOME/.kube/config")

	fs.BoolVarP(&opts.DryRun, types.FlagNameDryRun, "d", opts.DryRun,
		"Print the generated k8s resources on the stdout, not actual execute. Always use in debug mode")

	fs.BoolVarP(&opts.RequireConfirmation, types.FlagNameRequireConfirmation, "r", opts.RequireConfirmation,
		"specifies values whether you need to confirm the upgrade. The default RequireConfirmation value is false.")

	fs.StringArrayVar(&opts.Sets, types.FlagNameSet, []string{},
		"Sets values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")

	fs.StringArrayVar(&opts.ValueFiles, types.FlagNameValueFiles, []string{},
		"specify values in a YAML file (can specify multiple)")

	fs.BoolVar(&opts.Force, types.FlagNameForce, opts.Force,
		"Forced upgrading the cloud components without waiting")

	fs.StringVar(&opts.Profile, types.FlagNameProfile, opts.Profile,
		"Sets profile on the command line. If '--values' is specified, this is ignored")

	fs.StringVar(&opts.ExternalHelmRoot, types.FlagNameExternalHelmRoot, opts.ExternalHelmRoot,
		"Add external helm root path to keadm")

	fs.BoolVar(&opts.ReuseValues, types.FlagNameReuseValues, false,
		"reuse the last release's values and merge in any overrides from the command line via --set and -f.")

	fs.BoolVar(&opts.PrintFinalValues, types.FlagNamePrintFinalValues, false,
		"Print the final values configuration for debuging")

	fs.StringVar(&opts.ImageRepository, types.FlagNameImageRepository, opts.ImageRepository,
		"Choose a container image repository to pull the image of the kubedge component.")
}
