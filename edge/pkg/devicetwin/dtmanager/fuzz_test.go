package dtmanager

import (
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"testing"
	
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
)

var fuzzActions = map[int]string{
	0:  "dealTwinUpdate",
	1:  "dealTwinGet",
	2:  "dealTwinSync",
	3:  "dealDeviceAttrUpdate",
	4:  "dealDeviceStateUpdate",
	5:  "dealSendToCloud",
	6:  "dealSendToEdge",
	7:  "dealLifeCycle",
	8:  "dealConfirm",
	9:  "dealMembershipGet",
	10: "dealMembershipUpdate",
	11: "dealMembershipDetail",
}

func FuzzDeal(f *testing.F) {
	f.Fuzz(func(t *testing.T, device string, content []byte, actionType uint8) {
		msg := &model.Message{
			Content: content,
		}
		context, _ := dtcontext.InitDTContext()
		switch fuzzActions[int(actionType)%len(fuzzActions)] {
		case "dealTwinUpdate":
			dealTwinUpdate(context, device, msg)
		case "dealTwinGet":
			dealTwinGet(context, device, msg)
		case "dealTwinSync":
			dealTwinSync(context, device, msg)
		case "dealDeviceAttrUpdate":
			dealDeviceAttrUpdate(context, device, msg)
		case "dealDeviceStateUpdate":
			dealDeviceStateUpdate(context, device, msg)
		case "dealSendToCloud":
			dealSendToCloud(context, device, msg)
		case "dealSendToEdge":
			dealSendToEdge(context, device, msg)
		case "dealLifeCycle":
			dealLifeCycle(context, device, msg)
		case "dealConfirm":
			dealConfirm(context, device, msg)
		case "dealMembershipGet":
			dealMembershipGet(context, device, msg)
		case "dealMembershipUpdate":
			dealMembershipUpdate(context, device, msg)
		case "dealMembershipDetail":
			dealMembershipDetail(context, device, msg)
		}
	})
}
