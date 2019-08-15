package app

import (
	"github.com/spf13/cobra"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/edged"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
)

func NewEdgeSiteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "edgesite",
		Long: `EdgeSite helps running lightweight clusters at edge, which contains three modules: edgecontroller,
metamanager, and edged. EdgeController is an extended kubernetes controller which manages edge nodes and pods metadata 
so that the data can be targeted to a specific edge node. MetaManager is the message processor between edged and edgehub. 
It is also responsible for storing/retrieving metadata to/from a lightweight database (SQLite).Edged is an agent that 
runs on edge nodes and manages containerized applications.`,
		Run: func(cmd *cobra.Command, args []string) {
			registerModules()
			// start all modules
			core.Run()
		},
	}

	return cmd
}

// registerModules register all the modules started in edgesite
func registerModules() {
	edged.Register()
	edgecontroller.Register()
	metamanager.Register()
	dbm.InitDBManager()
}
