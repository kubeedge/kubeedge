package edgehub

import (
	"fmt"
	"github.com/spf13/cobra"
)

func NewEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable EdgeHub on specified edge nodes",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("EdgeHub enable command invoked (TODO)")
		},
	}
}
