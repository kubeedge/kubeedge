package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RuleSpec, defines rule of message delivery.
type RuleSpec struct {
	// Source, representing where the messages come from. Its value is the same with rule-endpoint name.
	// For example, rest or eventbus.
	Source string `json:"source"`
	// SourceResource is a map representing the resource info of source. For rest
	// rule-endpoint type its value is {"path":"/a/b"}. For eventbus rule-endpoint type its
	// value is {"topic":"<user define string>","node_name":"xxxx"}
	SourceResource map[string]string `json:"sourceResource"`
	// Target, representing where the messages go to. its value is the same with rule-endpoint name.
	// For example, eventbus or api or servicebus.
	Target string `json:"target"`
	// targetResource is a map representing the resource info of target. For api
	// rule-endpoint type its value is {"resource":"http://a.com"}. For eventbus rule-endpoint
	// type its value is {"topic":"/xxxx"}. For servicebus rule-endpoint type its value is {"path":"/request_path"}.
	TargetResource map[string]string `json:"targetResource"`
}

// RuleStatus, defines status of message delivery.
type RuleStatus struct {
	// SuccessMessages, success count of message delivery of rule.
	SuccessMessages int64 `json:"successMessages"`
	// FailMessages, failed count of message delivery of rule.
	FailMessages int64 `json:"failMessages"`
	// Errors, failed reasons of message delivery of rule.
	Errors []string `json:"errors"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Rule is the Schema for the rules API
// +k8s:openapi-gen=true
type Rule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuleSpec   `json:"spec"`
	Status RuleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// RuleList contains a list of Rule
type RuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rule `json:"items"`
}

// RuleEndpointSpec, defines endpoint of rule.
type RuleEndpointSpec struct {
	// RuleEndpointType, defines type: servicebus, rest
	RuleEndpointType string `json:"ruleEndpointType"`
	// Properties: properties of endpoint. for example:
	// servicebus:
	// {"service_port":"8080"}
	Properties map[string]string `json:"properties,omitempty"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// RuleEndpoint is the Schema for the rule-endpoints API
// +k8s:openapi-gen=true
type RuleEndpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RuleEndpointSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// RuleEndpointList contains a list of RuleEndpoint
type RuleEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuleEndpoint `json:"items"`
}
