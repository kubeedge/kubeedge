package dttype

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
)

//UnmarshalBaseMessage Unmarshal get
func UnmarshalBaseMessage(payload []byte) (*BaseMessage, error) {
	var get BaseMessage
	err := json.Unmarshal(payload, &get)
	if err != nil {
		return nil, err
	}
	return &get, nil
}

//BuildMembershipGetResult build memebership
func BuildMembershipGetResult(devices []*v1alpha2.Device) ([]byte, error) {
	payload, err := json.Marshal(devices)
	if err != nil {
		return []byte(""), err
	}
	return payload, nil
}

//BuildDeviceTwinResult build device twin result, 0:get,1:update,2:sync
func BuildDeviceTwinResult(deviceID string, twins []v1alpha2.Twin, dealType int) ([]byte, error) {
	result := v1alpha2.Device{}
	s := strings.Split(deviceID, "/")
	result.Namespace = s[0]
	result.Name = s[1]

	if dealType == 0 {
		for _, twin := range twins {
			if twin.Desired.Metadata != nil && strings.Compare(twin.Desired.Metadata["type"], "deleted") == 0 {
				continue
			}
			result.Status.Twins = append(result.Status.Twins, twin)
		}
	} else {
		result.Status.Twins = twins
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return []byte(""), err
	}
	return payload, nil
}

//BuildDeviceTwinDelta  build device twin delta
func BuildDeviceTwinDelta(twins []v1alpha2.Twin) ([]byte, bool) {
	result := v1alpha2.Device{}
	for _, v := range twins {
		if (v.Desired.Metadata != nil && strings.Compare(v.Desired.Metadata["type"], "deleted") == 0) || (v.Reported.Metadata != nil && strings.Compare(v.Reported.Metadata["type"], "deleted") == 0) {
			continue
		}

		if reflect.DeepEqual(v.Desired, v1alpha2.TwinProperty{}) {
			continue
		}

		if !reflect.DeepEqual(v.Reported, v1alpha2.TwinProperty{}) {
			if v.Desired.Value != v.Reported.Value && v.Desired.Value != "" {
				twin := v.DeepCopy()
				result.Status.Twins = append(result.Status.Twins, *twin)
			}
		} else {
			if v.Desired.Value != "" {
				twin := v.DeepCopy()
				result.Status.Twins = append(result.Status.Twins, *twin)
			}
		}
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return []byte(""), false
	}
	if len(result.Status.Twins) > 0 {
		return payload, true
	}
	return payload, false
}
