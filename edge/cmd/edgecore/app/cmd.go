package app

import (
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

func NewCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "edgecore",
		Short: "edgecore: is the core edge part of KubeEdge",
		Long: dedent.Dedent(`
			    ┌──────────────────────────────────────────────────────────┐
			    │ edgecore                                                 │
			    │ the core edge part of KubeEdge                           │
			    └──────────────────────────────────────────────────────────┘

			Example usage:

			    ┌──────────────────────────────────────────────────────────┐
			    │ Get current version:                                     │
			    ├──────────────────────────────────────────────────────────┤
			    │ # edgecore --version									   │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ Create default config:                                   │
			    ├──────────────────────────────────────────────────────────┤
			    │ # edgecore config default                                │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ run edgecore :                                           │
			    ├──────────────────────────────────────────────────────────┤
			    │ # edgecore core &                                        │
			    └──────────────────────────────────────────────────────────┘

		`),
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
		},
	}
	cmd.ResetFlags()

	cmd.AddCommand(NewEdgeCoreCommand())
	cmd.AddCommand(NewConfig())
	return cmd
}
