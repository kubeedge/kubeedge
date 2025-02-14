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

package message

import (
	"errors"
	"strings"

	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/common/constants"
)

const (
	OperationUpdateNodeTaskStatus = "UpdateNodeTaskStatus"
)

// TaskDownstreamMessage defines the downstream message content of the node task.
type TaskDownstreamMessage struct {
	Name     string `json:"name"`
	NodeName string `json:"nodeName"`
	Spec     any    `json:"spec"`
}

// Resource defines the message resource of the node task.
type Resource struct {
	// APIVersion defines the group/version of the node task resource
	APIVersion string
	// ResourceType defines the node task resource of Kubernetes.(e.g., nodeupgradejob, imagepulljob, etc.)
	ResourceType string
	// TaskName defines the name of the node task resource.
	TaskName string
	// Node defines the name of the node.
	Node string
}

// Check checks the resource fields.
func (r Resource) Check() error {
	if r.APIVersion == "" {
		return errors.New("the APIVersion field must not be blank")
	}
	if r.ResourceType == "" {
		return errors.New("the ResourceType field must not be blank")
	}
	if r.TaskName == "" {
		return errors.New("the TaskName field must not be blank")
	}
	if r.Node == "" {
		return errors.New("the Node field must not be blank")
	}
	return nil
}

// String returns resource that satisfy the message resource format
// {apiversion}/{resource_type}/{task_name}/node/{node_name}.
// It is best to use Check method to verify fields first.
func (r Resource) String() string {
	return strings.Join([]string{r.APIVersion, r.ResourceType, r.TaskName, "node", r.Node}, constants.ResourceSep)
}

// IsNodeTaskResource returns whether the resource is a node task resource.
func IsNodeTaskResource(r string) bool {
	return strings.HasPrefix(r, operationsv1alpha2.SchemeGroupVersion.String())
}

// ParseResource parse the node task resource from the message resource.
// It is best to use IsResource function to judge first:
//
//	if IsNodeTaskResource(resstr) {
//		res := ParseResource(resstr)
//	}
func ParseResource(resource string) *Resource {
	parts := strings.Split(resource, constants.ResourceSep)
	if len(parts) != 6 {
		klog.Warningf("invalid nodetask resource format: %s", resource)
		return nil
	}
	return &Resource{
		APIVersion:   parts[0] + "/" + parts[1], // {group}/{version}
		ResourceType: parts[2],
		TaskName:     parts[3],
		Node:         parts[5],
	}
}
