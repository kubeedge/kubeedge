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

package actions

import (
	"context"
	"fmt"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	nodetaskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

// runners is a global map variables,
// used to cache the implementation of the task action runner.
var runners = map[string]ActionRunner{}

func Init() {
	registerRunner(operationsv1alpha2.ResourceNodeUpgradeJob,
		newNodeUpgradeJobRunner())
	registerRunner(operationsv1alpha2.ResourceImagePrePullJob,
		newImagePrepullJobRunner())
}

// registerRunner registers the implementation of the task action runner.
func registerRunner(name string, runner ActionRunner) {
	runners[name] = runner
}

// GetRunner returns the implementation of the task action runner.
func GetRunner(name string) ActionRunner {
	return runners[name]
}

// ActionRunner defines the interface of the task action runner.
type ActionRunner interface {
	RunAction(startupAction string, task *nodetaskmsg.TaskDownstreamMessage)
}

// ActionFun defines the function type of the task action handler.
// The first return value defines whether the action should continue.
// In some scenarios, we want the flow to be paused and continue it
// when triggered elsewhere.
type ActionFun = func(ctx context.Context, task *nodetaskmsg.TaskDownstreamMessage) (bool, error)

// baseActionRunner defines the abstruct of the task action runner.
// The implementation of ActionRunner must compose this structure.
type baseActionRunner struct {
	// actions defines the function implementation of each action.
	actions map[string]ActionFun
	// flow defines the action flow of node task.
	flow actionflow.Flow
	// reportFun uses to report status of node task. If the err is not nil,
	// the failure status needs to be reported.
	reportFun func(action, taskname, nodename string, err error)
}

// Add task action runner to runners.
func (b *baseActionRunner) addAction(action string, handler ActionFun) {
	b.actions[action] = handler
}

// Get task action runner from runners, returns error when not found.
func (b *baseActionRunner) mustGetAction(action string) (ActionFun, error) {
	actionFn, ok := b.actions[action]
	if !ok {
		return nil, fmt.Errorf("invalid task action %s", action)
	}
	return actionFn, nil
}

// RunAction runs the task action.
func (b *baseActionRunner) RunAction(startupAction string, msg *nodetaskmsg.TaskDownstreamMessage) {
	ctx := context.Background()
	for action := b.flow.Find(startupAction); action != nil && !action.Final(); {
		actionFn, err := b.mustGetAction(action.Name)
		if err != nil {
			b.reportFun(action.Name, msg.Name, msg.NodeName, err)
			return
		}
		doNext, err := actionFn(ctx, msg)
		b.reportFun(action.Name, msg.Name, msg.NodeName, err)
		if err != nil {
			action = action.Next(false)
			continue
		}
		if !doNext {
			break
		}
		action = action.Next(true)
	}
}
