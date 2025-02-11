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

package actionflow

import "github.com/kubeedge/api/apis/operations/v1alpha2"

// Action defines the action of node task.
type Action struct {
	Name           string
	NextSuccessful *Action
	NextFailure    *Action
}

// Next returns the next action according to the success flag.
func (a *Action) Next(success bool) *Action {
	if success {
		return a.NextSuccessful
	}
	return a.NextFailure
}

// IsFinal returns whether current action is the final action.
func (a *Action) IsFinal() bool {
	return a.NextSuccessful == nil && a.NextFailure == nil
}

// Flow defines the action flow of node task.
type Flow struct {
	First *Action
}

// Find returns the found action by name.
func (sf *Flow) Find(name string) *Action {
	if sf.First.Name == name {
		return sf.First
	}
	return doFind(name, sf.First)
}

// Using recursion to find a action by name.
func doFind(name string, act *Action) *Action {
	if act.Name == name {
		return act
	}
	if act.NextSuccessful != nil {
		if next := doFind(name, act.NextSuccessful); next != nil {
			return next
		}
	}
	if act.NextFailure != nil {
		return doFind(name, act.NextFailure)
	}
	return nil
}

var (
	// FlowNodeUpgradeJob defines the action flow of node upgrade job.
	FlowNodeUpgradeJob = initNodeUpgradeJobFlow()
	// FlowImagePrePullJob defines the action flow of image pre pull job.
	FlowImagePrePullJob = initImagePrePullJob()
)

// initNodeUpgradeJobFlow initializes the action flow of node upgrade job.
//
//	Check (--> WaitingConfirmation --> Confirm) --> BackUp
//	  --> Upgrade --> [If fails]-> RollBack
func initNodeUpgradeJobFlow() *Flow {
	check := &Action{Name: string(v1alpha2.NodeUpgradeJobActionCheck)}
	waitingConfirmation := &Action{Name: string(v1alpha2.NodeUpgradeJobActionWaitingConfirmation)}
	check.NextSuccessful = waitingConfirmation
	confirm := &Action{Name: string(v1alpha2.NodeUpgradeJobActionConfirm)}
	waitingConfirmation.NextSuccessful = confirm
	backUp := &Action{Name: string(v1alpha2.NodeUpgradeJobActionBackUp)}
	confirm.NextSuccessful = backUp
	upgrade := &Action{Name: string(v1alpha2.NodeUpgradeJobActionUpgrade)}
	backUp.NextSuccessful = upgrade
	rollBack := &Action{Name: string(v1alpha2.NodeUpgradeJobActionRollBack)}
	upgrade.NextFailure = rollBack
	return &Flow{
		First: check,
	}
}

// initImagePrePullJob initializes the action flow of image pre pull job.
//
//	Check --> Pulls
func initImagePrePullJob() *Flow {
	check := &Action{Name: string(v1alpha2.ImagePrePullJobActionCheck)}
	pulls := &Action{Name: string(v1alpha2.ImagePrePullJobActionPull)}
	check.NextSuccessful = pulls
	return &Flow{
		First: check,
	}
}
