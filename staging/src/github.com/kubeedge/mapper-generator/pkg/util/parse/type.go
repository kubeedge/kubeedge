package parse

import (
	"fmt"
	"github.com/kubeedge/mapper-generator/pkg/common"
	dmiapi "github.com/kubeedge/mapper-generator/pkg/temp"
)

func ConvTwinsToGrpc(twins []common.Twin) ([]*dmiapi.Twin, error) {
	res := make([]*dmiapi.Twin, 0, len(twins))
	for _, twin := range twins {
		cur := &dmiapi.Twin{
			PropertyName: twin.PropertyName,
			Desired: &dmiapi.TwinProperty{
				Value: twin.Desired.Value,
				Metadata: map[string]string{
					"type":      twin.Desired.Metadatas.Type,
					"timestamp": twin.Desired.Metadatas.Timestamp,
				},
			},
			Reported: &dmiapi.TwinProperty{
				Value: twin.Reported.Value,
				Metadata: map[string]string{
					"type":      twin.Reported.Metadatas.Type,
					"timestamp": twin.Reported.Metadatas.Timestamp,
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
		desiredMeta := twin.Desired.GetMetadata()
		reportedMeta := twin.Reported.GetMetadata()
		cur := common.Twin{
			PropertyName: twin.GetPropertyName(),
			PVisitor:     srcTwin.PVisitor,
			Desired: common.DesiredData{
				Value: twin.Desired.GetValue(),
			},
			Reported: common.ReportedData{
				Value: twin.Reported.GetValue(),
			},
		}
		if desiredMeta != nil {
			cur.Desired.Metadatas = common.Metadata{
				Timestamp: twin.Desired.GetMetadata()["timestamp"],
				Type:      twin.Desired.GetMetadata()["type"],
			}
		}
		if reportedMeta != nil {
			cur.Reported.Metadatas = common.Metadata{
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
		}
		twins = append(twins, twinData)
	}

	return twins
}
