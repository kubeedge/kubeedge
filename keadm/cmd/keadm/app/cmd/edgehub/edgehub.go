package edgehub

import (
	"github.com/spf13/cobra"
)

func NewEdgeHubCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edgehub",
		Short: "Manage EdgeHub on edge nodes",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(NewEnableCmd())
	cmd.AddCommand(NewDisableCmd())

	return cmd
}
