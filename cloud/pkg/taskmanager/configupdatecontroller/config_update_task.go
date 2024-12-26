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

package configupdatecontroller

import (
	"fmt"
	"time"

	fsmapi "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/api/apis/operations/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

func currentConfigUpdateNodeState(id, nodeName string) (fsmapi.State, error) {
	v, ok := cache.CacheMap.Load(id)
	if !ok {
		return "", fmt.Errorf("can not find task %s", id)
	}
	task := v.(*v1alpha1.ConfigUpdateJob)
	var state fsmapi.State
	for _, status := range task.Status.Status {
		if status.NodeName == nodeName {
			state = status.State
			break
		}
	}
	if state == "" {
		state = fsmapi.TaskInit
	}
	return state, nil
}

func updateConfigUpdateNodeState(id, nodeName string, state fsmapi.State, event fsm.Event) error {
	v, ok := cache.CacheMap.Load(id)
	if !ok {
		return fmt.Errorf("can not find task %s", id)
	}
	task := v.(*v1alpha1.ConfigUpdateJob)
	newTask := task.DeepCopy()
	status := newTask.Status.DeepCopy()
	for i, nodeStatus := range status.Status {
		if nodeStatus.NodeName == nodeName {
			status.Status[i] = v1alpha1.TaskStatus{
				NodeName: nodeName,
				State:    state,
				Event:    event.Type,
				Action:   event.Action,
				Time:     time.Now().UTC().Format(time.RFC3339),
				Reason:   event.Msg,
			}
			break
		}
	}
	err := patchStatus(newTask, *status, client.GetCRDClient())
	if err != nil {
		return err
	}
	return nil
}

func NewConfigUpdateNodeFSM(taskName, nodeName string) *fsm.FSM {
	fsm := &fsm.FSM{}
	return fsm.NodeName(nodeName).ID(taskName).Guard(fsmapi.ConfigUpdateRule).StageSequence(fsmapi.ConfigUpdateStageSequence).CurrentFunc(currentConfigUpdateNodeState).UpdateFunc(updateConfigUpdateNodeState)
}

func NewConfigUpdateTaskFSM(taskName string) *fsm.FSM {
	fsm := &fsm.FSM{}
	return fsm.ID(taskName).Guard(fsmapi.ConfigUpdateRule).StageSequence(fsmapi.ConfigUpdateStageSequence).CurrentFunc(currentConfigUpdateTaskState).UpdateFunc(updateConfigUpdateTaskState)
}

func currentConfigUpdateTaskState(id, _ string) (fsmapi.State, error) {
	v, ok := cache.CacheMap.Load(id)
	if !ok {
		return "", fmt.Errorf("can not find task %s", id)
	}
	task := v.(*v1alpha1.ConfigUpdateJob)
	state := task.Status.State
	if state == "" {
		state = fsmapi.TaskInit
	}
	return state, nil
}

func updateConfigUpdateTaskState(id, _ string, state fsmapi.State, event fsm.Event) error {
	v, ok := cache.CacheMap.Load(id)
	if !ok {
		return fmt.Errorf("can not find task %s", id)
	}
	task := v.(*v1alpha1.ConfigUpdateJob)
	newTask := task.DeepCopy()
	status := newTask.Status.DeepCopy()

	status.Event = event.Type
	status.Action = event.Action
	status.Reason = event.Msg
	status.State = state
	status.Time = time.Now().UTC().Format(time.RFC3339)

	err := patchStatus(newTask, *status, client.GetCRDClient())

	if err != nil {
		return err
	}
	return nil
}
