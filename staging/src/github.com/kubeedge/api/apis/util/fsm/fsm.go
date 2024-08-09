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

package fsm

import (
	"fmt"

	"k8s.io/klog/v2"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
)

type FSM struct {
	id            string
	nodeName      string
	lastState     api.State
	currentFunc   func(id, nodeName string) (api.State, error)
	updateFunc    func(id, nodeName string, state api.State, event Event) error
	guard         map[string]api.State
	stageSequence map[api.State]api.State
}

func (F *FSM) NodeName(nodeName string) *FSM {
	F.nodeName = nodeName
	return F
}

type Event struct {
	Type            string
	Action          api.Action
	Msg             string
	ExternalMessage string
}

func (e Event) UniqueName() string {
	return e.Type + "/" + string(e.Action)
}

func (F *FSM) ID(id string) *FSM {
	F.id = id
	return F
}

func (F *FSM) LastState(lastState api.State) {
	F.lastState = lastState
}

func (F *FSM) CurrentFunc(currentFunc func(id, nodeName string) (api.State, error)) *FSM {
	F.currentFunc = currentFunc
	return F
}

func (F *FSM) UpdateFunc(updateFunc func(id, nodeName string, state api.State, event Event) error) *FSM {
	F.updateFunc = updateFunc
	return F
}

func (F *FSM) Guard(guard map[string]api.State) *FSM {
	F.guard = guard
	return F
}

func (F *FSM) StageSequence(stageSequence map[api.State]api.State) *FSM {
	F.stageSequence = stageSequence
	return F
}

func (F *FSM) CurrentState() (api.State, error) {
	if F.currentFunc == nil {
		return "", fmt.Errorf("currentFunc is nil")
	}
	return F.currentFunc(F.id, F.nodeName)
}

func (F *FSM) transitCheck(event Event) (api.State, api.State, error) {
	currentState, err := F.CurrentState()
	if err != nil {
		return "", "", err
	}
	if F.guard == nil {
		return "", "", fmt.Errorf("guard is nil ")
	}
	nextState, ok := F.guard[string(currentState)+"/"+event.UniqueName()]
	if !ok {
		return "", "", fmt.Errorf(string(currentState)+"/"+event.UniqueName(), " unsupported event")
	}
	return currentState, nextState, nil
}

func (F *FSM) AllowTransit(event Event) error {
	_, _, err := F.transitCheck(event)
	return err
}

func (F *FSM) Transit(event Event) error {
	currentState, nextState, err := F.transitCheck(event)
	if err != nil {
		return err
	}
	if F.updateFunc == nil {
		return fmt.Errorf("updateFunc is nil")
	}
	err = F.updateFunc(F.id, F.nodeName, nextState, event)
	if err != nil {
		return err
	}
	F.lastState = currentState
	return nil
}

func TaskFinish(state api.State) bool {
	return state == api.TaskFailed || state == api.TaskSuccessful
}

func (F *FSM) TaskStagCompleted(state api.State) bool {
	currentState, err := F.CurrentState()
	if err != nil {
		klog.Errorf("get %s current state failed: %s", F.id, err.Error())
		return false
	}
	if F.stageSequence[currentState] == state || TaskFinish(state) {
		return true
	}
	return false
}
