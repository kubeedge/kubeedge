package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/cloud"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/edge"
)

const (
	upgradeLongDescription = `Upgrade components of the cloud or the edge.
Specify whether to upgrade the cloud or the edge through three-level commands.`
)

func NewUpgradeCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade components of the cloud or the edge",
		Long:  upgradeLongDescription,
	}

	cmds.AddCommand(cloud.NewCloudUpgrade())
	cmds.AddCommand(edge.NewEdgeUpgrade())

	return cmds
}
