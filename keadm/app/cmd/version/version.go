package version

import (
	commonversion "github.com/kubeedge/kubeedge/version"

	"github.com/spf13/cobra"
)

var (
	versionLongDescription = `
"kubeedge version" command show detail version info
`
)

// NewEdgeJoin returns KubeEdge edge join command.
func NewVersionInfo() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Version info",
		Long:  versionLongDescription,
		Run: func(cmd *cobra.Command, args []string) {
			commonversion.Print()
		},
	}

	return cmd
}
