/*
Copyright 2021 The KubeEdge Authors.

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

package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/cmd/iptablesManager/app/options"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/iptables"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util/flag"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

func NewIptablesManagerCommand() *cobra.Command {
	opts := options.NewIptablesManagerOptions()
	cmd := &cobra.Command{
		Use:  "iptables",
		Long: `IptablesManager from KubeEdge on the cloudside`,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			flag.PrintFlags(cmd.Flags())

			// init kubeedge client
			var kubeAPIConfig v1alpha1.KubeAPIConfig
			kubeAPIConfig.KubeConfig = opts.KubeConfig
			client.InitKubeEdgeClient(&kubeAPIConfig)

			gis := informers.GetInformersManager()
			// The external mode will share the host network, forward to the inner stream port of the cloudcore.
			go iptables.NewIptablesManager().Run()
			gis.Start(beehiveContext.Done())

			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM,
				syscall.SIGQUIT, syscall.SIGILL, syscall.SIGTRAP, syscall.SIGABRT)
			s := <-c
			klog.Infof("Failed to run IptablesManager, os signal %v", s.String())
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
