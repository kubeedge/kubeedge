package app

import (
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edgesite",
		Short: "edgesite: edgesite helps running lightweight clusters at edge",
		Long: dedent.Dedent(`
			    ┌──────────────────────────────────────────────────────────┐
			    │ edgesite                                                 │
			    │ edgesite helps running lightweight clusters at edge      │
			    └──────────────────────────────────────────────────────────┘

			Example usage:
   			    ┌──────────────────────────────────────────────────────────┐
			    │ Get current version:	                                   │
			    ├──────────────────────────────────────────────────────────┤
			    │ # edgesite --version									   │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ Create default config:                                   │
			    ├──────────────────────────────────────────────────────────┤
			    │ # edgesite defaultconfig                                 │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ run edgesite :                                           │
			    ├──────────────────────────────────────────────────────────┤
			    │ # edgesite run &                                         │
			    └──────────────────────────────────────────────────────────┘

		`),
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
		},
	}

	cmd.ResetFlags()

	cmd.AddCommand(NewEdgeSiteCommand())
	cmd.AddCommand(NewDefaultConfig())
	return cmd
}
