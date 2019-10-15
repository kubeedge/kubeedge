package app

import (
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "edgecore",
		Short: "edgecore: is the core edge part of KubeEdge",
		Long: dedent.Dedent(`
			    ┌──────────────────────────────────────────────────────────┐
			    │ edgecore                                                │
			    │ the core edge part of KubeEdge                          │
			    └──────────────────────────────────────────────────────────┘

			Example usage:

			    ┌──────────────────────────────────────────────────────────┐
			    │ Create default config:                                   │
			    ├──────────────────────────────────────────────────────────┤
			    │ # edgecore defaultconfig                                │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ run edgecore :                                          │
			    ├──────────────────────────────────────────────────────────┤
			    │ # edgecore core &                                       │
			    └──────────────────────────────────────────────────────────┘

		`),
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
	cmd.ResetFlags()

	cmd.AddCommand(NewEdgeCoreCommand())
	cmd.AddCommand(NewConfig())
	return cmd
}
