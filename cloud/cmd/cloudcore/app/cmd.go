package app

import (
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/pkg/version/verflag"
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
			    │ Get current version:	                                   │
			    ├──────────────────────────────────────────────────────────┤
			    │ # cloudcore --version									   │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ Create default config:                                   │
			    ├──────────────────────────────────────────────────────────┤
			    │ # cloudcore defaultconfig                                │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ run cloudcore :                                          │
			    ├──────────────────────────────────────────────────────────┤
			    │ # cloudcore run &                                        │
			    └──────────────────────────────────────────────────────────┘

		`),
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
		},
	}

	cmd.ResetFlags()

	cmd.AddCommand(NewCloudCoreCommand())
	cmd.AddCommand(NewDefaultConfig())
	return cmd
}
