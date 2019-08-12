package app

import (
	"github.com/spf13/cobra"

	admissioncontroller "github.com/kubeedge/kubeedge/cloud/pkg/admissioncontroller"
)

func NewAdmissionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "admission",
		Long: `Admission is a auxiliary module which leverage the feature of Dynamic Admission Control from kubernetes, start the module
if any admission control against some kubeedge resource is wanted.`,
		Run: func(cmd *cobra.Command, args []string) {
			admissioncontroller.Run()
		},
	}

	return cmd
}
