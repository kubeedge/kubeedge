package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/pkg/version"
)

const (
	// DefaultErrorExitCode defines exit the code for failed action generally
	DefaultErrorExitCode = 1
)

func NewCmdVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of keadm",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return RunVersion(cmd)
		},
	}
	cmd.Flags().StringP("output", "o", "", "Output format; available options are 'yaml', 'json' and 'short'")
	return cmd
}

// RunVersion provides the version information of keadm in format depending on arguments
// specified in cobra.Command.
func RunVersion(cmd *cobra.Command) error {
	v := version.Get()

	const flag = "output"
	of, err := cmd.Flags().GetString(flag)
	if err != nil {
		return fmt.Errorf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
	}

	switch of {
	case "":
		fmt.Printf("version: %#v\n", v)
	case "short":
		fmt.Printf("%s\n", v)
	case "yaml":
		y, err := yaml.Marshal(&v)
		if err != nil {
			return err
		}
		fmt.Println(string(y))
	case "json":
		y, err := json.MarshalIndent(&v, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(y))
	default:
		return fmt.Errorf("invalid output format: %s", of)
	}

	return nil
}
