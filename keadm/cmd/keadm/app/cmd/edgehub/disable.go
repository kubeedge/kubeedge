package edgehub

import (
	"fmt"
	"github.com/spf13/cobra"
)

func NewDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable EdgeHub on specified edge nodes",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("EdgeHub disable command invoked (TODO)")
		},
	}
}
