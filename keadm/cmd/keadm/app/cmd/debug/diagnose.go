package debug

import (
	constant "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/spf13/cobra"
	"io"
)

var (
	edgeDiagnoseLongDescription = `"keadm debug collect " command obtain all the data of the current node
and then provide it to the operation and maintenance personnel to locate the problem
`
	edgeDiagnoseShortDescription = `Obtain all the data of the current node`

	edgeDiagnoseExample = `
# Check all items and specified as the current directory
keadm debug collect --path .
`
)

type Diagnose types.DiagnoseObject

// NewEdgeCollect returns KubeEdge edge debug collect command.
func NewDiagnose(out io.Writer, diagnoseOptions *types.DiagnoseOptions) *cobra.Command {
	if diagnoseOptions == nil {
		diagnoseOptions = newDiagnoseOptions()
	}
	cmd := &cobra.Command{
		Use:     "diagnose",
		Short:   edgeDiagnoseShortDescription,
		Long:    edgeDiagnoseLongDescription,
		Example: edgeDiagnoseExample,
	}
	for _, v := range constant.DiagnoseObjectMap {
		cmd.AddCommand(NewSubDiagnose(out, Diagnose(v)))
	}
	//addCollectOtherFlags(cmd, collectOptions)
	return cmd
}

// newCollectOptions returns a struct ready for being used for creating cmd collect flags.
func newDiagnoseOptions() *types.DiagnoseOptions {
	opts := &types.DiagnoseOptions{}
	return opts
}

//Start to collect data
func ExecuteDiagnose() error {
	return nil
}

func NewSubDiagnose(out io.Writer, object Diagnose) *cobra.Command {
	co := NewDiagnoseOptins()
	cmd := &cobra.Command{
		Short: object.Desc,
		Use:   object.Use,
		RunE: func(cmd *cobra.Command, args []string) error {
			return object.ExecuteDiagnose(object.Use, co)
		},
	}
	switch object.Use {

	}
	return cmd
}

// add flags
func NewDiagnoseOptins() *types.DiagnoseOptions {
	do := &types.DiagnoseOptions{}
	return do
}

func (da Diagnose) ExecuteDiagnose(use string, ops *types.DiagnoseOptions) error {
	switch use {
	case constant.ArgDiagnoseNode:
		DiagnoseNode()
	}
	return nil
}

func DiagnoseNode() {

}
