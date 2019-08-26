package app

import (
	"flag"

	"github.com/spf13/cobra"

	appConf "github.com/kubeedge/kubeedge/cloud/cmd/admission/app/options"
	"github.com/kubeedge/kubeedge/cloud/pkg/admissioncontroller"
)

func NewAdmissionCommand() *cobra.Command {
	config := appConf.NewConfig()
	cmd := &cobra.Command{
		Use: "admission",
		Long: `Admission leverage the feature of Dynamic Admission Control from kubernetes, start it
if want to admission control some kubeedge resources.`,
		Run: func(cmd *cobra.Command, args []string) {
			admissioncontroller.Run(config)
		},
	}

	config.AddFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return cmd
}
