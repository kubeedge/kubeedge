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
	"os/exec"

	"k8s.io/klog/v2"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
	"github.com/kubeedge/kubeedge/pkg/version"
)

func rollbackNode(taskReq commontypes.NodeTaskRequest) (event fsm.Event) {
	event = fsm.Event{
		Type:   "Rollback",
		Action: api.ActionSuccess,
	}
	var err error
	defer func() {
		if err != nil {
			event.Action = api.ActionFailure
			event.Msg = err.Error()
		}
	}()

	var upgradeReq *commontypes.NodeUpgradeJobRequest
	upgradeReq, err = getTaskRequest(taskReq)
	if err != nil {
		return
	}

	err = rollback(upgradeReq)
	if err != nil {
		return
	}
	return event
}

func rollback(upgradeReq *commontypes.NodeUpgradeJobRequest) error {
	klog.Infof("Begin to run rollback command")
	rollBackCmd := fmt.Sprintf("keadm rollback edge --name %s --history %s >> /tmp/keadm.log 2>&1",
		upgradeReq.UpgradeID, version.Get())

	// run upgrade cmd to upgrade edge node
	// use nohup command to start a child progress
	command := fmt.Sprintf("nohup %s &", rollBackCmd)
	cmd := exec.Command("bash", "-c", command)
	s, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run rollback command %s failed: %v, %s", command, err, s)
	}
	klog.Infof("!!! Finish rollback ")
	return nil
}
