package debug

import (
	"fmt"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/spf13/cobra"
	"io"
)

var (
	edgecollectLongDescription = `"keadm debug collect " command obtain all the data of the current node  
and then provide it to the operation and maintenance personnel to locate the problem
`
	edgecollectShortDescription = `Obtain all the data of the current node`

	edgecollectExample = `
# Check all items and specified as the current directory
keadm debug collect --path .
`
)

// NewEdgeCollect returns KubeEdge edge debug collect command.
func NewEdgeCollect(out io.Writer, collectOptions *types.ColletcOptions) *cobra.Command {
	if collectOptions == nil {
		collectOptions = newCollectOptions()
	}
	cmd := &cobra.Command{
		Use:     "collect",
		Short:   edgecollectShortDescription,
		Long:    edgecollectLongDescription,
		Example: edgecollectExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteCollect()
		},
	}

	addCollectOtherFlags(cmd, collectOptions)

	return cmd

}

// add Collect flags
func addCollectOtherFlags(cmd *cobra.Command, collectOptions *types.ColletcOptions) {
	cmd.Flags().StringVar(&collectOptions.Config, types.EdgecoreConfig, collectOptions.Config,
		fmt.Sprintf("Specify configuration file, defalut is %s", common.EdgecoreConfigPath))
	cmd.Flags().StringVar(&collectOptions.Detail, "detail", collectOptions.Detail,
		"Whether to print internal log output")
	cmd.Flags().StringVar(&collectOptions.OutputPath, "output-path", collectOptions.OutputPath,
		"Cache data and store data compression packages in a directory that default to the current directory")
}

// newCollectOptions returns a struct ready for being used for creating cmd collect flags.
func newCollectOptions() *types.ColletcOptions {
	opts := &types.ColletcOptions{}
	return opts
}

//Start to collect data
func ExecuteCollect() error {
	fmt.Println("Start collecting data")
	return nil
}
