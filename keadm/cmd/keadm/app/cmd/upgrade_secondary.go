/*
Copyright 2025 The KubeEdge Authors.

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

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/cloud"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/edge"
)

// NewUpgradeCommand creates a upgrade command instance and returns it.
func NewUpgradeCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade components of the cloud or the edge",
	}
	// Register three-level commands
	cmds.AddCommand(edge.NewUpgradeCommand())
	cmds.AddCommand(cloud.NewCloudUpgrade())
	return cmds
}

// NewBackupCommand creates a backup command instance and returns it.
func NewBackupCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "backup",
		Short: "Backup components of the cloud or the edge",
	}
	cmds.AddCommand(edge.NewBackupCommand())
	return cmds
}

// NewRollbackCommand creates a rollback command instance and returns it.
func NewRollbackCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback components of the cloud or the edge",
	}
	cmds.AddCommand(edge.NewRollbackCommand())
	return cmds
}
