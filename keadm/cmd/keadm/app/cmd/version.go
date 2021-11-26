package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/pkg/version"
)

func newCmdVersion(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of keadm",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunVersion(out, cmd)
		},
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringP("output", "o", "", "Output format; available options are 'yaml', 'json' and 'short'")
	return cmd
}

// RunVersion provides the version information of keadm in format depending on arguments
// specified in cobra.Command.
func RunVersion(out io.Writer, cmd *cobra.Command) error {
	v := version.Get()

	const flag = "output"
	of, err := cmd.Flags().GetString(flag)
	if err != nil {
		return fmt.Errorf("error accessing flag %s for command %s", flag, cmd.Name())
	}

	switch of {
	case "":
		fmt.Fprintf(out, "version: %#v\n", v)
	case "short":
		fmt.Fprintf(out, "%s\n", v)
	case "yaml":
		y, err := yaml.Marshal(&v)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(y))
	case "json":
		y, err := json.MarshalIndent(&v, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(y))
	default:
		return fmt.Errorf("invalid output format: %s", of)
	}

	return nil
}
