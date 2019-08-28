package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apiserver/pkg/util/term"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin"
	"github.com/kubeedge/kubeedge/edge/pkg/edged"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	"github.com/kubeedge/kubeedge/edge/pkg/servicebus"
	"github.com/kubeedge/kubeedge/edge/test"
	edgemesh "github.com/kubeedge/kubeedge/edgemesh/pkg"
	"github.com/kubeedge/kubeedge/pkg/util/flag"
	"github.com/kubeedge/kubeedge/pkg/version"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

// NewEdgeCoreCommand create edgecore cmd
func NewEdgeCoreCommand() *cobra.Command {
	opts := options.NewEdgeCoreOptions()
	cmd := &cobra.Command{
		Use: "edgecore",
		Long: `Edgecore is the core edge part of KubeEdge, which contains six modules: devicetwin, edged, 
edgehub, eventbus, metamanager, and servicebus. DeviceTwin is responsible for storing device status 
and syncing device status to the cloud. It also provides query interfaces for applications. Edged is an 
agent that runs on edge nodes and manages containerized applications and devices. Edgehub is a web socket 
client responsible for interacting with Cloud Service for the edge computing (like Edge Controller as in the KubeEdge 
Architecture). This includes syncing cloud-side resource updates to the edge, and reporting 
edge-side host and device status changes to the cloud. EventBus is a MQTT client to interact with MQTT 
servers (mosquito), offering publish and subscribe capabilities to other components. MetaManager 
is the message processor between edged and edgehub. It is also responsible for storing/retrieving metadata 
to/from a lightweight database (SQLite).ServiceBus is a HTTP client to interact with HTTP servers (REST), 
offering HTTP client capabilities to components of cloud to reach HTTP servers running at edge. `,
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

// registerModules register all the modules started in edgecore
func registerModules() {
	devicetwin.Register()
	edged.Register()
	edgehub.Register()
	eventbus.Register()
	edgemesh.Register()
	metamanager.Register()
	servicebus.Register()
	test.Register()
	dbm.InitDBManager()
}
