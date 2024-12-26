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

package taskexecutor

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"k8s.io/klog/v2"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
	"github.com/kubeedge/kubeedge/pkg/version"
)

const (
	TaskConfigUpdate = "configUpdate"
)

type ConfigUpdate struct {
	*BaseExecutor
}

func (c *ConfigUpdate) Name() string {
	return c.name
}

func NewConfigUpdateExecutor() Executor {
	methods := map[string]func(types.NodeTaskRequest) fsm.Event{
		string(api.TaskChecking):     configCheck,
		string(api.TaskInit):         emptyInit,
		"":                           emptyInit,
		string(api.BackingUpState):   backupNode,
		string(api.RollingBackState): rollbackNode,
		string(api.UpdatingState):    update,
	}
	return &ConfigUpdate{
		BaseExecutor: NewBaseExecutor(TaskConfigUpdate, methods),
	}
}

func configCheck(_ types.NodeTaskRequest) (event fsm.Event) {
	// todo: validation update configuration
	return fsm.Event{
		Type:   "Check",
		Action: api.ActionSuccess,
	}
}

func update(taskReq types.NodeTaskRequest) (event fsm.Event) {
	event = fsm.Event{
		Type: TaskConfigUpdate,
	}
	updateReq, err := getConfigUpdateTaskRequest(taskReq)
	if err != nil {
		event.Action = api.ActionFailure
		event.Msg = err.Error()
		return
	}
	var setFields string
	for updateKey, updateVal := range updateReq.UpdateFields {
		setFields = setFields + fmt.Sprintf("%s=%s,", updateKey, updateVal)
	}
	setFields = strings.TrimSuffix(setFields, ",")
	err = keadmUpdate(*updateReq, setFields)
	if err != nil {
		event.Action = api.ActionFailure
		event.Msg = err.Error()
	}
	return
}

func keadmUpdate(updateReq types.ConfigUpdateJobRequest, setFields string) error {
	configUpdateCmd := fmt.Sprintf("keadm config-update --nodeVersion %s --updateID %s --set %s > /tmp/keadm.log 2>&1",
		version.Get(), updateReq.UpdateID, setFields)
	klog.Infof("Begin to run update command %s", configUpdateCmd)

	// run upgrade cmd to upgrade edge node
	// use nohup command to start a child progress
	command := fmt.Sprintf("nohup %s &", configUpdateCmd)
	cmd := exec.Command("bash", "-c", command)
	s, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run config update command %s failed: %v, %s", command, err, s)
	}
	klog.Infof("!!! Finish config update task")
	return nil
}

func getConfigUpdateTaskRequest(taskReq types.NodeTaskRequest) (*types.ConfigUpdateJobRequest, error) {
	data, err := json.Marshal(taskReq.Item)
	if err != nil {
		return nil, err
	}
	var configUpdateReq types.ConfigUpdateJobRequest
	err = json.Unmarshal(data, &configUpdateReq)
	if err != nil {
		return nil, err
	}
	return &configUpdateReq, err
}
