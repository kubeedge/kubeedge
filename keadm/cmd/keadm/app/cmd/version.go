package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
		Run: func(cmd *cobra.Command, args []string) {
			err := RunVersion(cmd)
			CheckErr(err, fatal)
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
		fmt.Printf("error accessing flag %s for command %s: %v\n", flag, cmd.Name(), err)
		os.Exit(1)
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
		fmt.Printf(string(y))
	case "json":
		y, err := json.MarshalIndent(&v, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf(string(y))
	default:
		return fmt.Errorf("invalid output format: %s", of)
	}

	return nil
}

// fatal prints the message if set and then exits.
func fatal(msg string, code int) {
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}

		fmt.Fprint(os.Stderr, msg)
	}
	os.Exit(code)
}

// CheckErr formats a given error as a string and calls the passed handleErr
// func with that string and an exit code.
func CheckErr(err error, handleErr func(string, int)) {
	switch err.(type) {
	case nil:
		return
	default:
		handleErr(err.Error(), DefaultErrorExitCode)
	}
}
