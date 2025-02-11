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

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

type NodeJob interface {
	Name() string
	ResourceType() string
	Concurrency() int
	Spec() any
	Tasks() []NodeJobTask
	GetObject() any
}

type NodeJobTask interface {
	NodeName() string
	CanExecute() bool
	// Status returns the status of the node task.
	Status() operationsv1alpha2.NodeTaskStatus
	// SetStatus sets the status of the node task.
	SetStatus(status operationsv1alpha2.NodeTaskStatus)
	// Action returns the current action of the node task.
	Action() (*actionflow.Action, error)
	// SetAction sets the action of the node task.
	SetAction(action *actionflow.Action)
	// GetObject returns the node task object.
	GetObject() any
}

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
