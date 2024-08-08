package parse

import (
	"fmt"

	dmiapi "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/common"
)

func ConvTwinsToGrpc(twins []common.Twin) ([]*dmiapi.Twin, error) {
	res := make([]*dmiapi.Twin, 0, len(twins))
	for _, twin := range twins {
		cur := &dmiapi.Twin{
			PropertyName: twin.PropertyName,
			ObservedDesired: &dmiapi.TwinProperty{
				Value: twin.ObservedDesired.Value,
				Metadata: map[string]string{
					"type":      twin.ObservedDesired.Metadata.Type,
					"timestamp": twin.ObservedDesired.Metadata.Timestamp,
				},
			},
			Reported: &dmiapi.TwinProperty{
				Value: twin.Reported.Value,
				Metadata: map[string]string{
					"type":      twin.Reported.Metadata.Type,
					"timestamp": twin.Reported.Metadata.Timestamp,
				},
			},
		}
		res = append(res, cur)
	}
	return res, nil
}

func ConvGrpcToTwins(twins []*dmiapi.Twin, srcTwins []common.Twin) ([]common.Twin, error) {
	res := make([]common.Twin, 0, len(twins))
	for _, twin := range twins {
		var srcTwin common.Twin
		for _, found := range srcTwins {
			if twin.GetPropertyName() == found.PropertyName {
				srcTwin = found
				break
			}
		}
		if srcTwin.PropertyName == "" {
			return nil, fmt.Errorf("not found src twin name %s while update status", twin.GetPropertyName())
		}
		desiredMeta := twin.ObservedDesired.GetMetadata()
		reportedMeta := twin.Reported.GetMetadata()
		cur := common.Twin{
			PropertyName: twin.GetPropertyName(),
			Property:     srcTwin.Property,
			ObservedDesired: common.TwinProperty{
				Value: twin.ObservedDesired.GetValue(),
			},
			Reported: common.TwinProperty{
				Value: twin.Reported.GetValue(),
			},
		}
		if desiredMeta != nil {
			cur.ObservedDesired.Metadata = common.Metadata{
				Timestamp: twin.ObservedDesired.GetMetadata()["timestamp"],
				Type:      twin.ObservedDesired.GetMetadata()["type"],
			}
		}
		if reportedMeta != nil {
			cur.Reported.Metadata = common.Metadata{
				Timestamp: twin.Reported.GetMetadata()["timestamp"],
				Type:      twin.Reported.GetMetadata()["type"],
			}
		}
		res = append(res, cur)
	}
	return res, nil
}

func ConvMsgTwinToGrpc(msgTwin map[string]*common.MsgTwin) []*dmiapi.Twin {
	var twins []*dmiapi.Twin
	for name, twin := range msgTwin {
		twinData := &dmiapi.Twin{
			PropertyName: name,
			Reported: &dmiapi.TwinProperty{
				Value: *twin.Actual.Value,
				Metadata: map[string]string{
					"type":      twin.Metadata.Type,
					"timestamp": twin.Actual.Metadata.Timestamp,
				}},
			ObservedDesired: &dmiapi.TwinProperty{
				Value: *twin.Expected.Value,
				Metadata: map[string]string{
					"type":      twin.Metadata.Type,
					"timestamp": twin.Actual.Metadata.Timestamp,
				}},
		}
		twins = append(twins, twinData)
	}

	return twins
}
