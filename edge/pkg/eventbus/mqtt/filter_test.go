package mqtt

import (
	"testing"

	"github.com/256dpi/gomqtt/packet"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func init() {
	RegisterMsgHandler()

	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	add := &common.ModuleInfo{
		ModuleName: modules.TwinGroup,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(add)
	beehiveContext.AddModuleGroup(modules.DeviceTwinModuleName, modules.TwinGroup)
}

func TestDispatch(t *testing.T) {
	msg := packet.Message{
		Topic:   "$hw/events/device/sampledevice/twin/test",
		Payload: []byte("device sample"),
		QOS:     0,
		Retain:  false,
	}
	NewMessageMux().Dispatch(msg.Topic, msg.Payload)
	message, _ := beehiveContext.Receive(modules.DeviceTwinModuleName)

	t.Run("SuccessDispatchDeviceTwinMsg", func(t *testing.T) {
		want := messagepkg.OperationResponse
		if message.GetOperation() != want {
			t.Errorf("Wrong message received : Wanted operation: %v and Got operation: %v", want, message.GetOperation())
		}
	})
}
