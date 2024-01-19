package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/cloud"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/edge"
)

const (
	upgradeLongDescription = `Upgrade components of the cloud or the edge.
Specify whether to upgrade the cloud or the edge through three-level commands.
If no three-level command, it upgrades edge components.`
)

func NewUpgradeCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade components of the cloud or the edge",
		Long:  upgradeLongDescription,
	}

	edgecmd := edge.NewEdgeUpgrade()

	// Used for backward compatibility of the edgecore trigger the upgrade command
	upgradeOptions := edge.NewUpgradeOptions()
	edge.AddUpgradeFlags(cmds, upgradeOptions)
	cmds.RunE = edgecmd.RunE

	// Register three-level commands
	cmds.AddCommand(edgecmd)
	cmds.AddCommand(cloud.NewCloudUpgrade())
	return cmds
}
