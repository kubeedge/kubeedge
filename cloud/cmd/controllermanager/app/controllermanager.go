package app

import (
	"context"
	"flag"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	_ "sigs.k8s.io/controller-runtime/pkg/client/config" // using to set --kubeconfig flag
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeedge/kubeedge/cloud/cmd/controllermanager/app/options"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager"
	"github.com/kubeedge/kubeedge/pkg/features"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

func NewControllerManagerCommand(ctx context.Context) *cobra.Command {
	log.SetLogger(klog.FromContext(ctx))
	opts := options.NewControllerManagerOptions()
	cmd := &cobra.Command{
		Use:  "controller-manager",
		Long: `The node group controller manager run a bunch of controllers`,
		Run: func(_cmd *cobra.Command, _args []string) {
			for _, fg := range opts.FeatureGates {
				if err := features.DefaultMutableFeatureGate.Set(fmt.Sprintf("%s=true", fg)); err != nil {
					klog.Errorf("failed to set feature gate '%s', err: %v", fg, err)
				}
			}
			kubeconfig := controllerruntime.GetConfigOrDie()
			mgr, err := controllermanager.NewControllerManager(ctx, kubeconfig, opts.HealthProbeBindAddress)
			if err != nil {
				klog.Fatalf("failed to get controller manager, %v", err)
			}

			// mgr.Start will block until the manager has stopped
			if err := mgr.Start(ctx); err != nil {
				klog.Fatalf("failed to start controller manager, %v", err)
			}
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
