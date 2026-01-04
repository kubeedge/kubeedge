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

package fsm

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
)

func TestFSM_Builder(t *testing.T) {
	fsm := new(FSM)
	id := "test-id"
	nodeName := "test-node"
	lastState := api.State("initial")
	guard := map[string]api.State{"start": "running"}
	stageSeq := map[api.State]api.State{"step1": "step2"}
	currentFunc := func(id, nodeName string) (api.State, error) { return "current", nil }
	updateFunc := func(id, nodeName string, state api.State, event Event) error { return nil }

	fsm.ID(id).
		NodeName(nodeName).
		LastState(lastState)

	fsm.Guard(guard).
		StageSequence(stageSeq).
		CurrentFunc(currentFunc).
		UpdateFunc(updateFunc)

	// Since fields are private, we can only verify through behavior or if we had getters.
	// However, we can verify CurrentState works which relies on CurrentFunc, ID and NodeName
	state, err := fsm.CurrentState()
	assert.NoError(t, err)
	assert.Equal(t, api.State("current"), state)
}

func TestFSM_CurrentState(t *testing.T) {
	tests := []struct {
		name        string
		currentFunc func(id, nodeName string) (api.State, error)
		wantState   api.State
		wantErr     bool
	}{
		{
			name: "Success",
			currentFunc: func(id, nodeName string) (api.State, error) {
				return "running", nil
			},
			wantState: "running",
			wantErr:   false,
		},
		{
			name: "Error from function",
			currentFunc: func(id, nodeName string) (api.State, error) {
				return "", errors.New("db error")
			},
			wantState: "",
			wantErr:   true,
		},
		{
			name:        "Nil function",
			currentFunc: nil,
			wantState:   "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsm := new(FSM)
			if tt.currentFunc != nil {
				fsm.CurrentFunc(tt.currentFunc)
			}
			state, err := fsm.CurrentState()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantState, state)
			}
		})
	}
}

func TestFSM_AllowTransit(t *testing.T) {
	// Setup common states and events
	const (
		StateIdle    api.State = "Idle"
		StateRunning api.State = "Running"
		StateFailed  api.State = "Failed"
	)

	eventStart := Event{Type: "Task", Action: "Start"}
	eventStop := Event{Type: "Task", Action: "Stop"}

	// Define Guard: Idle + Task/Start -> Running
	guard := map[string]api.State{
		string(StateIdle) + "/" + eventStart.UniqueName(): StateRunning,
	}

	tests := []struct {
		name         string
		currentState api.State
		event        Event
		wantErr      bool
	}{
		{
			name:         "Valid transition",
			currentState: StateIdle,
			event:        eventStart,
			wantErr:      false,
		},
		{
			name:         "Invalid transition (no guard rule)",
			currentState: StateRunning,
			event:        eventStart,
			wantErr:      true,
		},
		{
			name:         "Invalid event",
			currentState: StateIdle,
			event:        eventStop,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsm := new(FSM)
			fsm.Guard(guard)
			fsm.CurrentFunc(func(id, nodeName string) (api.State, error) {
				return tt.currentState, nil
			})

			err := fsm.AllowTransit(tt.event)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFSM_Transit(t *testing.T) {
	const (
		StateIdle    api.State = "Idle"
		StateRunning api.State = "Running"
	)
	eventStart := Event{Type: "Task", Action: "Start"}

	tests := []struct {
		name         string
		currentState api.State
		event        Event
		updateFunc   func(id, nodeName string, state api.State, event Event) error
		wantErr      bool
	}{
		{
			name:         "Successful transit",
			currentState: StateIdle,
			event:        eventStart,
			updateFunc: func(id, nodeName string, state api.State, event Event) error {
				if state != StateRunning {
					return errors.New("wrong next state")
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:         "Update function fails",
			currentState: StateIdle,
			event:        eventStart,
			updateFunc: func(id, nodeName string, state api.State, event Event) error {
				return errors.New("update failed")
			},
			wantErr: true,
		},
		{
			name:         "Missing update function",
			currentState: StateIdle,
			event:        eventStart,
			updateFunc:   nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsm := new(FSM)
			// Define Guard: Idle + Task/Start -> Running
			fsm.Guard(map[string]api.State{
				string(StateIdle) + "/" + eventStart.UniqueName(): StateRunning,
			})
			fsm.CurrentFunc(func(id, nodeName string) (api.State, error) {
				return tt.currentState, nil
			})
			if tt.updateFunc != nil {
				fsm.UpdateFunc(tt.updateFunc)
			}

			err := fsm.Transit(tt.event)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTaskFinish(t *testing.T) {
	assert.True(t, TaskFinish(api.TaskFailed))
	assert.True(t, TaskFinish(api.TaskSuccessful))
	assert.False(t, TaskFinish(api.State("Running")))
}

func TestFSM_TaskStagCompleted(t *testing.T) {
	const (
		StateStep1 api.State = "Step1"
		StateStep2 api.State = "Step2"
	)

	stageSeq := map[api.State]api.State{
		StateStep1: StateStep2,
	}

	tests := []struct {
		name         string
		currentState api.State
		targetState  api.State
		want         bool
	}{
		{
			name:         "Sequence matches",
			currentState: StateStep1,
			targetState:  StateStep2,
			want:         true,
		},
		{
			name:         "Sequence does not match",
			currentState: StateStep2,
			targetState:  StateStep1,
			want:         false,
		},
		{
			name:         "Target is finish state (Success)",
			currentState: StateStep1,
			targetState:  api.TaskSuccessful,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsm := new(FSM).StageSequence(stageSeq)
			fsm.CurrentFunc(func(id, nodeName string) (api.State, error) {
				return tt.currentState, nil
			})

			got := fsm.TaskStagCompleted(tt.targetState)
			assert.Equal(t, tt.want, got)
		})
	}
}
