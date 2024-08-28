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
	UpgradingState State = "Upgrading"
)

// CurrentState/Event/Action: NextState
var UpgradeRule = map[string]State{
	"Init/Init/Success":    TaskChecking,
	"Init/Init/Failure":    TaskFailed,
	"Init/TimeOut/Failure": TaskFailed,
	"Init/Upgrade/Success": TaskSuccessful,

	"Checking/Check/Success":   BackingUpState,
	"Checking/Check/Failure":   TaskFailed,
	"Checking/TimeOut/Failure": TaskFailed,

	"BackingUp/Backup/Success":  UpgradingState,
	"BackingUp/Backup/Failure":  TaskFailed,
	"BackingUp/TimeOut/Failure": TaskFailed,

	"Upgrading/Upgrade/Success": TaskSuccessful,
	"Upgrading/Upgrade/Failure": TaskFailed,
	"Upgrading/TimeOut/Failure": TaskFailed,

	// TODO provide options for task failure, such as successful node upgrade rollback.
	"RollingBack/Rollback/Failure": TaskFailed,
	"RollingBack/TimeOut/Failure":  TaskFailed,
	"RollingBack/Rollback/Success": TaskFailed,

	"Upgrading/Rollback/Failure": TaskFailed,
	"Upgrading/Rollback/Success": TaskFailed,

	//TODO delete in version 1.18
	"Init/Rollback/Failure": TaskFailed,
	"Init/Rollback/Success": TaskFailed,
}

var UpdateStageSequence = map[State]State{
	"":             TaskChecking,
	TaskInit:       TaskChecking,
	TaskChecking:   BackingUpState,
	BackingUpState: UpgradingState,
	UpgradingState: RollingBackState,
}
