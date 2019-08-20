package app

import (
	"github.com/spf13/cobra"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller"
)

func NewCloudCoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "cloudcore",
		Long: `CloudCore is the core cloud part of KubeEdge, which contains three modules: cloudhub,
edgecontroller, and devicecontroller. Cloudhub is a web server responsible for watching changes at the cloud side,
caching and sending messages to EdgeHub. EdgeController is an extended kubernetes controller which manages 
edge nodes and pods metadata so that the data can be targeted to a specific edge node. DeviceController is an extended 
kubernetes controller which manages devices so that the device metadata/status date can be synced between edge and cloud.`,
		Run: func(cmd *cobra.Command, args []string) {
			registerModules()
			// start all modules
			core.Run()
		},
	}

	cmd.AddCommand(NewCmdVersion())
	return cmd
}

// registerModules register all the modules started in cloudcore
func registerModules() {
	cloudhub.Register()
	edgecontroller.Register()
	devicecontroller.Register()
}
