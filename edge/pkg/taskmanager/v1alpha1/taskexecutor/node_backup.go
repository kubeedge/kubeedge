/*
Copyright 2023 The KubeEdge Authors.

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

package taskexecutor

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/common/constants"
	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/util/files"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
	"github.com/kubeedge/kubeedge/pkg/version"
)

func backupNode(commontypes.NodeTaskRequest) (event fsm.Event) {
	event = fsm.Event{
		Type:   "Backup",
		Action: api.ActionSuccess,
	}
	var err error
	defer func() {
		if err != nil {
			event.Action = api.ActionFailure
			event.Msg = err.Error()
		}
	}()
	backupPath := filepath.Join(common.KubeEdgeBackupPath, version.Get().String())
	err = backup(backupPath)
	if err != nil {
		cleanErr := os.Remove(backupPath)
		if cleanErr != nil {
			klog.Warningf("clean backup path failed: %s", err.Error())
		}
		return
	}
	return event
}

func backup(backupPath string) error {
	config := options.GetEdgeCoreConfig()
	klog.Infof("backup start, backup path: %s", backupPath)
	if err := os.MkdirAll(backupPath, 0750); err != nil {
		return fmt.Errorf("mkdirall failed: %v", err)
	}

	// backup edgecore.db: copy from origin path to backup path
	if err := files.FileCopy(config.DataBase.DataSource, filepath.Join(backupPath, "edgecore.db")); err != nil {
		return fmt.Errorf("failed to backup db: %v", err)
	}
	// backup edgecore.yaml: copy from origin path to backup path
	if err := files.FileCopy(constants.DefaultConfigDir+"edgecore.yaml", filepath.Join(backupPath, "edgecore.yaml")); err != nil {
		return fmt.Errorf("failed to back config: %v", err)
	}
	// backup edgecore: copy from origin path to backup path
	if err := files.FileCopy(filepath.Join(constants.KubeEdgeUsrBinPath, constants.KubeEdgeBinaryName),
		filepath.Join(backupPath, constants.KubeEdgeBinaryName)); err != nil {
		return fmt.Errorf("failed to backup edgecore: %v", err)
	}
	return nil
}
