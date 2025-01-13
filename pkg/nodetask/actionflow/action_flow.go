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

// Final returns whether current action is the final action.
func (a *Action) Final() bool {
	return a.NextSuccessful == nil && a.NextFailure == nil
}

// Flow defines the action flow of node task.
type Flow struct {
	First Action
}

// Find returns the found action by name.
func (sf *Flow) Find(name string) *Action {
	if sf.First.Name == name {
		return &sf.First
	}
	return doFind(name, &sf.First)
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
	//
	//	Init --> Check (--> Confirm --> WaitingConfirmation) -
	//    --> BackUp --> Upgrade
	//	                    └─ [If fails]-> RollBack
	FlowNodeUpgradeJob = &Flow{
		First: Action{
			Name: string(v1alpha2.NodeUpgradeJobActionInit),
			NextSuccessful: &Action{
				Name: string(v1alpha2.NodeUpgradeJobActionCheck),
				NextSuccessful: &Action{
					Name: string(v1alpha2.NodeUpgradeJobActionConfirm),
					NextSuccessful: &Action{
						Name: string(v1alpha2.NodeUpgradeJobActionWaitingConfirmation),
						NextSuccessful: &Action{
							Name: string(v1alpha2.NodeUpgradeJobActionBackUp),
							NextSuccessful: &Action{
								Name: string(v1alpha2.NodeUpgradeJobActionUpgrade),
								NextFailure: &Action{
									Name: string(v1alpha2.NodeUpgradeJobActionRollBack),
								},
							},
						},
					},
				},
			},
		},
	}

	// FlowImagePrePullJob defines the action flow of image pre pull job.
	//
	//	Init --> Check --> Pulls
	FlowImagePrePullJob = &Flow{
		First: Action{
			Name: string(v1alpha2.ImagePrePullJobActionInit),
			NextSuccessful: &Action{
				Name: string(v1alpha2.ImagePrePullJobActionCheck),
				NextSuccessful: &Action{
					Name: string(v1alpha2.ImagePrePullJobActionPull),
				},
			},
		},
	}
)
