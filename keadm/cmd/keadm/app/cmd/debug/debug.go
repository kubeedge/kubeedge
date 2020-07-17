package debug

import (
	"io"

	"github.com/spf13/cobra"
)

// NewCmdDebug represents the debug command
func NewCmdDebug(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "keadm help command provide debug function to help diagnose the cluster",
	}
	cmd.AddCommand(NewCmdDebugGet(out))
	return cmd
}
