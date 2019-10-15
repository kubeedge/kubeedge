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
		Use:   "defaultconfig",
		Short: "create default config for cloudcore",
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
