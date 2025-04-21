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
	OperationUpdateNodeActionStatus = "UpdateNodeActionStatus"
)

// UpstreamMessage defines the upstream message content of the node job.
type UpstreamMessage struct {
	// Action defines the action of the node job.
	Action string `json:"action"`
	// Succ defines whether the action is successful.
	Succ bool `json:"succ"`
	// Reason defines error message.
	Reason string `json:"reason"`
	// Extend uses to stored serializable string. Some node actions may do multiple things,
	// this field can store the Extend infos for cloud parsing.
	Extend string `json:"extend"`
}

// Resource defines the message resource of the node job.
type Resource struct {
	// APIVersion defines the group/version of the node job resource
	APIVersion string
	// ResourceType defines the node job resource of Kubernetes.(e.g., nodeupgradejob, imagepulljob, etc.)
	ResourceType string
	// JobName defines the name of the node job job resource.
	JobName string
	// NodeName defines the name of the node.
	NodeName string
}

// Check checks the resource fields.
func (r Resource) Check() error {
	if r.APIVersion == "" {
		return errors.New("the APIVersion field must not be blank")
	}
	if r.ResourceType == "" {
		return errors.New("the ResourceType field must not be blank")
	}
	if r.JobName == "" {
		return errors.New("the job name field must not be blank")
	}
	if r.NodeName == "" {
		return errors.New("the node name field must not be blank")
	}
	return nil
}

// String returns resource that satisfy the message resource format
// {apiversion}/{resource_type}/{job_name}/node/{node_name}.
// It is best to use Check method to verify fields first.
func (r Resource) String() string {
	return strings.Join([]string{r.APIVersion, r.ResourceType, r.JobName, "node", r.NodeName}, constants.ResourceSep)
}

// IsNodeJobResource returns whether the resource is a node job resource.
func IsNodeJobResource(r string) bool {
	return strings.HasPrefix(r, operationsv1alpha2.SchemeGroupVersion.String())
}

// ParseResource parse the node job resource from the message resource.
// It is best to use IsResource function to judge first:
//
//	if IsNodeJobResource(resstr) {
//		res := ParseResource(resstr)
//	}
func ParseResource(resource string) Resource {
	parts := strings.Split(resource, constants.ResourceSep)
	if len(parts) != 6 {
		klog.Warningf("invalid nodetask resource format: %s", resource)
		return Resource{}
	}
	return Resource{
		APIVersion:   parts[0] + "/" + parts[1], // {group}/{version}
		ResourceType: parts[2],
		JobName:      parts[3],
		NodeName:     parts[5],
	}
}
