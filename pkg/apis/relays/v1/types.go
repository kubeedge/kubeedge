// pkg/relay/v1/types.go
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Relayrc is the Schema for the relayrcs API
type Relayrc struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RelayrcSpec   `json:"spec,omitempty"`
	Status RelayrcStatus `json:"status,omitempty"`
}

type NodeAddress struct {
	// Required
	IP string `json:"ip,omitempty"`
	// Required
	Port int64 `json:"port,omitempty"`
}

type RelayData struct {
	AddrData map[string]NodeAddress `json:"addrdata,omitempty"`
}

// RelayrcSpec defines the desired state of Relayrc
type RelayrcSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Open    bool      `json:"open,omitempty"`
	RelayID string    `json:"relayId,omitempty"`
	Data    RelayData `json:"data,omitempty"`
}

// RelayrcStatus defines the observed state of Relayrc
type RelayrcStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// RelayrcList contains a list of Relayrc
type RelayrcList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Relayrc `json:"items"`
}
