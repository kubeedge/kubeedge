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

package wrap

import (
	"fmt"
	"time"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

type NodeJob interface {
	// Name returns the name of the node job.
	Name() string
	// ResourceType returns the resource type of the node job.
	ResourceType() string
	// Concurrency returns the concurrency in the node job spec.
	Concurrency() int
	// Spec returns the spec of the node job.
	Spec() any
	// Tasks returns the node tasks of the node job.
	Tasks() []NodeJobTask
	// GetObject returns the node job object.
	GetObject() any
}

type NodeJobTask interface {
	// NodeName returns the node name of the node task.
	NodeName() string
	// CanExecute returns whether the node job status can be executed.
	CanExecute() bool
	// Phase returns the phase of the node task.
	Phase() operationsv1alpha2.NodeTaskPhase
	// ToSuccessful sets the phase of the node task to successful.
	ToSuccessful()
	// ToInProgress sets the phase of the node task to in progress, and set the processing time.
	ToInProgress(t time.Time)
	// ToFailure sets the phase of the node task to failure, and set the failure reason.
	ToFailure(reason string)
	// Action returns the current action of the node task. If the action value is empty,
	// return the first action from the action flow.
	Action() (*actionflow.Action, error)
	// SetAction sets the action of the node task.
	SetAction(action *actionflow.Action)
	// GetObject returns the node task object.
	GetObject() any
}

// WithEventObj returns the node job wrap based on the event object.
func WithEventObj(obj any) (NodeJob, error) {
	switch obj := obj.(type) {
	case *operationsv1alpha2.NodeUpgradeJob:
		return NewNodeUpgradeJob(obj), nil
	case *operationsv1alpha2.ImagePrePullJob:
		return NewImagePrepullJob(obj), nil
	default:
		return nil, fmt.Errorf("invalid event object type %T", obj)
	}
}
