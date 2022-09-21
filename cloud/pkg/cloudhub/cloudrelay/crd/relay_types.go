package crd

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeAddress struct {
	// Required
	IP string `json:"ip,omitempty"`
	// Required
	Port int64 `json:"port,omitempty"`
}

type RelayData struct {
	//
	AddrData map[string]NodeAddress `json:"addrdata,omitempty"`
}

func (rd *RelayData) MarshalJSON() ([]byte, error) {
	return json.Marshal(rd.AddrData)
}

func (rd *RelayData) UnmarshalJSON(data []byte) error {
	var out map[string]NodeAddress
	err := json.Unmarshal(data, &out)
	if err != nil {
		return err
	}
	rd.AddrData = out
	return nil
}

type RelaySpec struct {
	// Required
	RelaySwitch bool      `json:"relaySwitch,omitempty"`
	RelayID     string    `json:"relayId,omitempty"`
	Data        RelayData `json:"data,omitempty"`
}

type Relay struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RelaySpec `json:"spec,omitempty"`
}
