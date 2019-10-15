package app

import (
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "cloudcore",
		Short: "cloudcore: core cloud part of KubeEdge",
		Long: dedent.Dedent(`
			    ┌──────────────────────────────────────────────────────────┐
			    │ cloudcore                                                │
			    │ the core cloud part of KubeEdge                          │
			    └──────────────────────────────────────────────────────────┘

			Example usage:

			    ┌──────────────────────────────────────────────────────────┐
			    │ Create default config:                                   │
			    ├──────────────────────────────────────────────────────────┤
			    │ # cloudcore defaultconfig                                │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ run cloudcore :                                          │
			    ├──────────────────────────────────────────────────────────┤
			    │ # cloudcore core &                                       │
			    └──────────────────────────────────────────────────────────┘

		`),
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
	cmd.ResetFlags()

	cmd.AddCommand(NewCloudCoreCommand())
	cmd.AddCommand(NewConfig())
	return cmd
}
