package app

import (
	"fmt"

	"github.com/spf13/cobra"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/edged"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	"github.com/kubeedge/kubeedge/edgesite/cmd/edgesite/app/options"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgesite/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgesite/v1alpha1/validation"
	"github.com/kubeedge/kubeedge/pkg/util"
	"github.com/kubeedge/kubeedge/pkg/util/flag"
	"github.com/kubeedge/kubeedge/pkg/version"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

func NewEdgeSiteCommand() *cobra.Command {
	opts := options.NewEdgeSiteOptions()
	cmd := &cobra.Command{
		Use: "edgesite",
		Long: `EdgeSite helps running lightweight clusters at edge, which contains three modules: edgecontroller,
metamanager, and edged. EdgeController is an extended kubernetes controller which manages edge nodes and pods metadata
so that the data can be targeted to a specific edge node. MetaManager is the message processor between edged and edgehub.
It is also responsible for storing/retrieving metadata to/from a lightweight database (SQLite).Edged is an agent that
runs on edge nodes and manages containerized applications.`,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			flag.PrintMinConfigAndExitIfRequested(v1alpha1.NewMinEdgeSiteConfig())
			flag.PrintDefaultConfigAndExitIfRequested(v1alpha1.NewDefaultEdgeSiteConfig())
			flag.PrintFlags(cmd.Flags())

			if errs := opts.Validate(); len(errs) > 0 {
				klog.Fatal(util.SpliceErrors(errs))
			}

			config, err := opts.Config()
			if err != nil {
				klog.Fatal(err)
			}

			if errs := validation.ValidateEdgeSiteConfiguration(config); len(errs) > 0 {
				klog.Fatal(util.SpliceErrors(errs.ToAggregate().Errors()))
			}

			// To help debugging, immediately log version
			klog.Infof("Version: %+v", version.Get())

			registerModules(config)
			// start all modules
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

// registerModules register all the modules started in edgesite
func registerModules(c *v1alpha1.EdgeSiteConfig) {
	edged.Register(c.Modules.Edged)
	edgecontroller.Register(c.Modules.EdgeController, c.KubeAPIConfig, c.Modules.Edged.HostnameOverride, true)
	metamanager.Register(c.Modules.MetaManager)
	// Nodte: Need to put it to the end, and wait for all models to register before executing
	dbm.InitDBConfig(c.DataBase.DriverName, c.DataBase.AliasName, c.DataBase.DataSource)
}
