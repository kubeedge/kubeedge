package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apiserver/pkg/util/term"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/cmd/cloudcore/app/options"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller"
	"github.com/kubeedge/kubeedge/pkg/util/flag"
	"github.com/kubeedge/kubeedge/pkg/version"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

func NewCloudCoreCommand() *cobra.Command {
	opts := options.NewCloudCoreOptions()
	cmd := &cobra.Command{
		Use: "cloudcore",
		Long: `CloudCore is the core cloud part of KubeEdge, which contains three modules: cloudhub,
edgecontroller, and devicecontroller. Cloudhub is a web server responsible for watching changes at the cloud side,
caching and sending messages to EdgeHub. EdgeController is an extended kubernetes controller which manages 
edge nodes and pods metadata so that the data can be targeted to a specific edge node. DeviceController is an extended 
kubernetes controller which manages devices so that the device metadata/status date can be synced between edge and cloud.`,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			flag.PrintFlags(cmd.Flags())

			// To help debugging, immediately log version
			klog.Infof("Version: %+v", version.Get())
			registerModules()
			// start all modules
			core.Run()
		},
	}
	fs := cmd.Flags()
	namedFs := opts.Flags()
	verflag.AddFlags(namedFs.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFs.FlagSet("global"), cmd.Name())
	for _, f := range namedFs.FlagSets {
		fs.AddFlagSet(f)
	}

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFs, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFs, cols)
	})

	return cmd
}

// registerModules register all the modules started in cloudcore
func registerModules() {
	cloudhub.Register()
	edgecontroller.Register()
	devicecontroller.Register()
	synccontroller.Register()
}
