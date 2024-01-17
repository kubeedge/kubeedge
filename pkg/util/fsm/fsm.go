package fsm

import (
	"fmt"

	api "github.com/kubeedge/kubeedge/pkg/apis/fsm/v1alpha1"
)

type FSM struct {
	id          string
	lastState   api.State
	currentFunc func(id string) (api.State, error)
	updateFunc  func(id string, state api.State, event Event) error
	guard       map[string]api.State
}

type Event struct {
	Type   string
	Action api.Action
	Name   string
	Error  error
}

func (e Event) UniqueName() string {
	return e.Type + "/" + string(e.Action)
}

const ()

func (F *FSM) Id(id string) {
	F.id = id
}

func (F *FSM) LastState(lastState api.State) {
	F.lastState = lastState
}

func (F *FSM) CurrentFunc(currentFunc func(id string) (api.State, error)) {
	F.currentFunc = currentFunc
}

func (F *FSM) UpdateFunc(updateFunc func(id string, state api.State, event Event) error) {
	F.updateFunc = updateFunc
}

func (F *FSM) Guard(guard map[string]api.State) {
	F.guard = guard
}

func (F *FSM) CurrentState() (api.State, error) {
	if F.currentFunc == nil {
		return "", fmt.Errorf("currentFunc is nil")
	}
	return F.currentFunc(F.id)
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
		return "", "", fmt.Errorf("unsupported event")
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
	err = F.updateFunc(F.id, nextState, event)
	if err != nil {
		return err
	}
	F.lastState = currentState
	return nil
}
