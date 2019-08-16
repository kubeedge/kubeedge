package app

import (
	"github.com/spf13/cobra"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin"
	"github.com/kubeedge/kubeedge/edge/pkg/edged"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	"github.com/kubeedge/kubeedge/edge/pkg/servicebus"
	"github.com/kubeedge/kubeedge/edge/test"
	edgemesh "github.com/kubeedge/kubeedge/edgemesh/pkg"
)

// NewEdgeCoreCommand create edgecore cmd
func NewEdgeCoreCommand() *cobra.Command {
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
			registerModules()
			// start all modules
			core.Run()
		},
	}

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
