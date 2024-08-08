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

package v1alpha1

const (
	PullingState State = "Pulling"
)

// CurrentState/Event/Action: NextState
var PrePullRule = map[string]State{
	"Init/Init/Success":    TaskChecking,
	"Init/Init/Failure":    TaskFailed,
	"Init/TimeOut/Failure": TaskFailed,

	"Checking/Check/Success":   PullingState,
	"Checking/Check/Failure":   TaskFailed,
	"Checking/TimeOut/Failure": TaskFailed,

	"Pulling/Pull/Success":    TaskSuccessful,
	"Pulling/Pull/Failure":    TaskFailed,
	"Pulling/TimeOut/Failure": TaskFailed,
}

var PrePullStageSequence = map[State]State{
	"":           TaskChecking,
	TaskInit:     TaskChecking,
	TaskChecking: PullingState,
}
