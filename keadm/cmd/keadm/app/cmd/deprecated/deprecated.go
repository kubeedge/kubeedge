package deprecated

import (
	"github.com/spf13/cobra"
)

// NewDeprecated represents the deprecated command
func NewDeprecated() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deprecated",
		Short: "keadm deprecated command",
		Long:  `keadm deprecated command provides some subcommands that are deprecated in kubeedge installation, using this sub command is NOT recommended`,
	}

	cmd.ResetFlags()

	cmd.AddCommand(NewDeprecatedCloudInit())
	cmd.AddCommand(NewDeprecatedEdgeJoin())
	cmd.AddCommand(NewDeprecatedKubeEdgeReset())

	return cmd
}
