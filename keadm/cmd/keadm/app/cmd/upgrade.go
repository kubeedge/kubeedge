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

// NewUpgradeCommand creates a upgrade command instance and returns it.
func NewUpgradeCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade components of the cloud or the edge",
		Long:  upgradeLongDescription,
	}

	// Used for backward compatibility of the edgecore trigger the upgrade command
	upgradeOptions := edge.NewUpgradeOptions()
	cmds.RunE = func(_ *cobra.Command, _ []string) error {
		return upgradeOptions.Upgrade()
	}
	edge.AddUpgradeFlags(cmds, upgradeOptions)

	// Register three-level commands
	cmds.AddCommand(edge.NewEdgeUpgrade())
	cmds.AddCommand(cloud.NewCloudUpgrade())
	return cmds
}
