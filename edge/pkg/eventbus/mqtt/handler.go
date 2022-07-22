package mqtt

import (
	"encoding/base64"
	"fmt"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// handleDevice for topic "$hw/events/device/+/twin/+", "$hw/events/node/+/membership/get"
func handleDeviceTwin(topic string, payload []byte) {
	target := modules.TwinGroup
	resource := base64.URLEncoding.EncodeToString([]byte(topic))
	// routing key will be $hw.<project_id>.events.user.bus.response.cluster.<cluster_id>.node.<node_id>.<base64_topic>
	message := beehiveModel.NewMessage("").BuildRouter(modules.BusGroup, modules.UserGroup,
		resource, messagepkg.OperationResponse).FillBody(string(payload))
	klog.Info(fmt.Sprintf("Received msg from mqttserver, deliver to %s with resource %s", target, message.GetResource()))
	beehiveContext.SendToGroup(target, *message)
}

// handleUploadTopic for topic "SYS/dis/upload_records"
func handleUploadTopic(topic string, payload []byte) {
	target := modules.HubGroup
	message := beehiveModel.NewMessage("").BuildRouter(modules.BusGroup, modules.UserGroup,
		topic, beehiveModel.UploadOperation).FillBody(string(payload))
	klog.Info(fmt.Sprintf("Received msg from mqttserver, deliver to %s with resource %s", target, message.GetResource()))
	beehiveContext.SendToGroup(target, *message)
}
