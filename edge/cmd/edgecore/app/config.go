package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	kyaml "sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

func NewConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "create or get config for edgecore",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
	cmd.ResetFlags()

	cmd.AddCommand(NewDefaultConfig())
	return cmd
}

func NewDefaultConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "default",
		Short: "create default config for edgecore",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.NewDefaultEdgeCoreConfig()
			data, err := kyaml.Marshal(cfg)
			if err != nil {
				fmt.Printf("Marshal defaut config to yaml error %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%v\n", string(data))
			return nil
		},
		Args: cobra.NoArgs,
	}
	return cmd
}
