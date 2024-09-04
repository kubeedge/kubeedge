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

package imageprepullcontroller

import (
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	fsmapi "github.com/kubeedge/api/apis/fsm/v1alpha1"
	v1alpha12 "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/api/apis/operations/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

func currentPrePullNodeState(id, nodeName string) (v1alpha12.State, error) {
	v, ok := cache.CacheMap.Load(id)
	if !ok {
		return "", fmt.Errorf("can not find task %s", id)
	}
	task := v.(*v1alpha1.ImagePrePullJob)
	var state v1alpha12.State
	for _, status := range task.Status.Status {
		if status.NodeName == nodeName {
			state = status.State
			break
		}
	}
	if state == "" {
		state = v1alpha12.TaskInit
	}
	return state, nil
}

func updatePrePullNodeState(id, nodeName string, state v1alpha12.State, event fsm.Event) error {
	v, ok := cache.CacheMap.Load(id)
	if !ok {
		return fmt.Errorf("can not find task %s", id)
	}
	task := v.(*v1alpha1.ImagePrePullJob)
	newTask := task.DeepCopy()
	status := newTask.Status.DeepCopy()
	for i, nodeStatus := range status.Status {
		if nodeStatus.NodeName == nodeName {
			var imagesStatus []v1alpha1.ImageStatus
			err := json.Unmarshal([]byte(event.ExternalMessage), &imagesStatus)
			if err != nil {
				klog.Warningf("Failed to unmarshal images status: %v", err)
			}
			status.Status[i] = v1alpha1.ImagePrePullStatus{
				TaskStatus: &v1alpha1.TaskStatus{
					NodeName: nodeName,
					State:    state,
					Event:    event.Type,
					Action:   event.Action,
					Time:     time.Now().UTC().Format(time.RFC3339),
					Reason:   event.Msg,
				},
				ImageStatus: imagesStatus,
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

func NewImagePrePullNodeFSM(taskName, nodeName string) *fsm.FSM {
	fsm := &fsm.FSM{}
	return fsm.NodeName(nodeName).ID(taskName).Guard(fsmapi.PrePullRule).StageSequence(fsmapi.PrePullStageSequence).CurrentFunc(currentPrePullNodeState).UpdateFunc(updatePrePullNodeState)
}

func NewImagePrePullTaskFSM(taskName string) *fsm.FSM {
	fsm := &fsm.FSM{}
	return fsm.ID(taskName).Guard(fsmapi.PrePullRule).StageSequence(fsmapi.PrePullStageSequence).CurrentFunc(currentPrePullTaskState).UpdateFunc(updateUpgradeTaskState)
}

func currentPrePullTaskState(id, _ string) (v1alpha12.State, error) {
	v, ok := cache.CacheMap.Load(id)
	if !ok {
		return "", fmt.Errorf("can not find task %s", id)
	}
	task := v.(*v1alpha1.ImagePrePullJob)
	state := task.Status.State
	if state == "" {
		state = v1alpha12.TaskInit
	}
	return state, nil
}

func updateUpgradeTaskState(id, _ string, state v1alpha12.State, event fsm.Event) error {
	v, ok := cache.CacheMap.Load(id)
	if !ok {
		return fmt.Errorf("can not find task %s", id)
	}
	task := v.(*v1alpha1.ImagePrePullJob)
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
