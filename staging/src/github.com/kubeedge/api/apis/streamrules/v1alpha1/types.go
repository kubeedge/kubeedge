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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StreamTarget struct {
	EndpointRef string `json:"endpointRef"`
	// {"node_name":"xxx", "path": "/a/b"}
	TargetResource map[string]string `json:"targetResource"`
}

type StreamRuleSpec struct {
	Targets []StreamTarget `json:"targets,omitempty"`
}

type StreamRuleStatus struct {
	// SuccessMessages represents success count of message delivery of rule.
	SuccessMessages int64 `json:"successMessages"`
	// FailMessages represents failed count of message delivery of rule.
	FailMessages int64 `json:"failMessages"`
	// Errors represents failed reasons of message delivery of rule.
	Errors []string `json:"errors"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Rule is the Schema for the rules API
// +k8s:openapi-gen=true
type StreamRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StreamRuleSpec   `json:"spec"`
	Status StreamRuleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StreamRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StreamRule `json:"items"`
}

type StreamRuleEndpointSpec struct {
	Protocol   ProtocolType      `json:"protocol"`
	URL        string            `json:"url"`
	Properties map[string]string `json:"properties,omitempty"`
}

// ProtocolType indicates the protocol type of StreamRuleEndpoint
type ProtocolType string

// Protocol types
const (
	ProtocolTypeWebSocket ProtocolType = "websocket"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// RuleEndpoint is the Schema for the ruleendpoints API
// +k8s:openapi-gen=true
type StreamRuleEndpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec StreamRuleEndpointSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StreamRuleEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StreamRuleEndpoint `json:"items"`
}
