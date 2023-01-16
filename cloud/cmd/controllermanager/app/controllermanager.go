package app

import (
	"context"
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"

	// set --kubeconfig option
	_ "sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/kubeedge/kubeedge/cloud/cmd/controllermanager/app/options"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

func NewControllerManagerCommand(ctx context.Context) *cobra.Command {
	opts := options.NewControllerManagerOptions()
	cmd := &cobra.Command{
		Use:  "controller-manager",
		Long: `The node group controller manager run a bunch of controllers`,
		Run: func(cmd *cobra.Command, args []string) {
			Run(ctx)
		},
	}

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	fs := cmd.Flags()
	namedFs := opts.Flags()
	verflag.AddFlags(namedFs.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFs.FlagSet("global"), cmd.Name())
	for _, f := range namedFs.FlagSets {
		fs.AddFlagSet(f)
	}

	return cmd
}

func Run(ctx context.Context) {
	kubeconfig := controllerruntime.GetConfigOrDie()
	mgr, err := controllermanager.NewAppsControllerManager(ctx, kubeconfig)
	if err != nil {
		klog.Fatalf("failed to get controller manager, %v", err)
	}

	// mgr.Start will block until the manager has stopped
	if err := mgr.Start(ctx); err != nil {
		klog.Fatalf("failed to start controller manager, %v", err)
	}
}
