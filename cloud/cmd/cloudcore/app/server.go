package app

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/cmd/cloudcore/app/options"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller"
	kele "github.com/kubeedge/kubeedge/cloud/pkg/leaderelection"
	"github.com/kubeedge/kubeedge/cloud/pkg/synccontroller"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1/validation"
	"github.com/kubeedge/kubeedge/pkg/util"
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
			flag.PrintMinConfigAndExitIfRequested(v1alpha1.NewMinCloudCoreConfig())
			flag.PrintDefaultConfigAndExitIfRequested(v1alpha1.NewDefaultCloudCoreConfig())
			flag.PrintFlags(cmd.Flags())

			if errs := opts.Validate(); len(errs) > 0 {
				klog.Fatal(util.SpliceErrors(errs))
			}

			config, err := opts.Config()
			if err != nil {
				klog.Fatal(err)
			}

			if errs := validation.ValidateCloudCoreConfiguration(config); len(errs) > 0 {
				klog.Fatal(util.SpliceErrors(errs.ToAggregate().Errors()))
			}

			// To help debugging, immediately log version
			klog.Infof("Version: %+v", version.Get())

			registerModules(config)

			// If leader election is enabled, runCommand via LeaderElector until done and exit.
			if config.LeaderElection.LeaderElect {
				electionChecker := kele.NewLeaderReadyzAdaptor(time.Second * 20)
				hubconfig.Config.Checker = electionChecker
				kele.Run(config, electionChecker)
				return
			}

			// Start all modules if disable leader election
			core.Run()
		},
	}
	fs := cmd.Flags()
	namedFs := opts.Flags()
	verflag.AddFlags(namedFs.FlagSet("global"))
	flag.AddFlags(namedFs.FlagSet("global"))
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
func registerModules(c *v1alpha1.CloudCoreConfig) {
	cloudhub.Register(c.Modules.CloudHub, c.KubeAPIConfig)
	edgecontroller.Register(c.Modules.EdgeController, c.KubeAPIConfig, "", false)
	devicecontroller.Register(c.Modules.DeviceController, c.KubeAPIConfig)
	synccontroller.Register(c.Modules.SyncController, c.KubeAPIConfig)
	cloudstream.Register(c.Modules.CloudStream)
}
