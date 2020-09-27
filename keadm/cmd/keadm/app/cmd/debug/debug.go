package debug

import (
	"io"

	"github.com/spf13/cobra"
)

var (
	edgeDebugLongDescription = `keadm debug command provide debug function to help diagnose the cluster`
)

// NewEdgeDebug returns KubeEdge edge debug command.
func NewEdgeDebug(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Provide debug function to help diagnose the cluster",
		Long:  edgeDebugLongDescription,
	}

	// add subCommand
	cmd.AddCommand(NewCmdDebugGet(out, nil))

	return cmd
}
