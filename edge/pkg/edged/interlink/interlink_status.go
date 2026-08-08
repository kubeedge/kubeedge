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

package interlink

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MapInterLinkStatusToPodPhase determines the Kubernetes PodPhase from
// the InterLink container status responses.
//
// Mapping rules:
//   - If any container is Waiting → PodPending
//   - If any container is Running → PodRunning
//   - If all containers are Terminated with exit code 0 → PodSucceeded
//   - If any container is Terminated with non-zero exit code → PodFailed
//   - If no containers reported → PodPending (awaiting first status)
func MapInterLinkStatusToPodPhase(containers []ContainerStatusResponse) v1.PodPhase {
	if len(containers) == 0 {
		return v1.PodPending
	}

	allTerminated := true
	anyFailed := false
	anyRunning := false
	anyWaiting := false

	for _, c := range containers {
		if c.State.Waiting != nil {
			anyWaiting = true
			allTerminated = false
		} else if c.State.Running != nil {
			anyRunning = true
			allTerminated = false
		} else if c.State.Terminated != nil {
			if c.State.Terminated.ExitCode != 0 {
				anyFailed = true
			}
		} else {
			// No state set, assume waiting
			anyWaiting = true
			allTerminated = false
		}
	}

	if anyWaiting {
		return v1.PodPending
	}
	if anyRunning {
		return v1.PodRunning
	}

	if allTerminated {
		if anyFailed {
			return v1.PodFailed
		}
		return v1.PodSucceeded
	}

	return v1.PodPending
}

// BuildPodStatusPatch constructs a JSON merge patch for updating a pod's status
// based on InterLink status responses. This patch is sent through KubeEdge's
// MetaManager → CloudHub pipeline to report HPC job status back to the cloud.
type PodStatusPatch struct {
	Status PodStatusPatchBody `json:"status"`
}

// PodStatusPatchBody contains the status fields to patch.
type PodStatusPatchBody struct {
	Phase      v1.PodPhase      `json:"phase"`
	Conditions []v1.PodCondition `json:"conditions,omitempty"`
	Message    string            `json:"message,omitempty"`
}

// NewPodStatusPatch creates a status patch from InterLink status responses.
func NewPodStatusPatch(podStatus PodStatusResponse) PodStatusPatch {
	phase := MapInterLinkStatusToPodPhase(podStatus.Containers)

	conditions := []v1.PodCondition{
		{
			Type:               "InterLinkManaged",
			Status:             v1.ConditionTrue,
			Reason:             "InterLinkOffloaded",
			Message:            "Pod is managed by InterLink for remote HPC/HTC execution",
			LastTransitionTime: metav1.Now(),
		},
	}

	var message string
	switch phase {
	case v1.PodPending:
		message = "Pod is queued or initializing on the remote HPC/HTC backend"
	case v1.PodRunning:
		message = "Pod is running on the remote HPC/HTC backend"
	case v1.PodSucceeded:
		message = "Pod completed successfully on the remote HPC/HTC backend"
	case v1.PodFailed:
		message = "Pod failed on the remote HPC/HTC backend"
	}

	return PodStatusPatch{
		Status: PodStatusPatchBody{
			Phase:      phase,
			Conditions: conditions,
			Message:    message,
		},
	}
}
