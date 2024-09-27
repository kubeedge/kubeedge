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

type State string

type Action string

const (
	NodeAvailable      State = "Available"
	NodeUpgrading      State = "Upgrading"
	NodeRollingBack    State = "RollingBack"
	NodeConfigUpdating State = "ConfigUpdating"
)

const (
	TaskInit       State = "Init"
	TaskChecking   State = "Checking"
	TaskSuccessful State = "Successful"
	TaskFailed     State = "Failed"
	TaskPause      State = "Pause"
)

const (
	ActionSuccess      Action = "Success"
	ActionFailure      Action = "Failure"
	ActionConfirmation Action = "Confirmation"
)

const (
	EventTimeOut = "TimeOut"
)
