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

package v1alpha2

type JobPhase string

// Constants for job phase
const (
	JobPhaseInit       JobPhase = "Init"
	JobPhaseInProgress JobPhase = "InProgress"
	JobPhaseCompleted  JobPhase = "Completed"
	JobPhaseFailure    JobPhase = "Failure"
)

type NodeTaskPhase string

// Constants for node task status.
const (
	NodeTaskPhasePending    NodeTaskPhase = "Pending"
	NodeTaskPhaseInProgress NodeTaskPhase = "InProgress"
	NodeTaskPhaseSuccessful NodeTaskPhase = "Successful"
	NodeTaskPhaseFailure    NodeTaskPhase = "Failure"
	NodeTaskPhaseUnknown    NodeTaskPhase = "Unknown"
)

// Constants for node job check items.
const (
	CheckItemCPU  string = "cpu"
	CheckItemMem  string = "mem"
	CheckItemDisk string = "disk"
)
