/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ctl

import (
	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/confirm"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/describe"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/edit"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/exec"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/get"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/logs"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/restart"
)

var ctlShortDescription = `Commands operating on the data plane at edge`

// NewCtl returns KubeEdge edge pod command.
func NewCtl() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ctl",
		Short: ctlShortDescription,
		Long:  ctlShortDescription,
	}

	cmd.AddCommand(get.NewEdgeGet())
	cmd.AddCommand(restart.NewEdgeRestart())
	cmd.AddCommand(confirm.NewEdgeConfirm())
	cmd.AddCommand(logs.NewEdgePodLogs())
	cmd.AddCommand(exec.NewEdgePodExec())
	cmd.AddCommand(describe.NewEdgeDescribe())
	cmd.AddCommand(edit.NewEdgeEdit())
	return cmd
}
