package processor

import (
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func init() {
	createKey := queryKey{
		operation: constants.CSIOperationTypeCreateVolume,
	}

	deleteKey := queryKey{
		operation: constants.CSIOperationTypeDeleteVolume,
	}
	publishKey := queryKey{
		operation: constants.CSIOperationTypeControllerPublishVolume,
	}
	unpublishKey := queryKey{
		operation: constants.CSIOperationTypeControllerUnpublishVolume,
	}

	processors[createKey] = &volumeProcessor{}
	processors[deleteKey] = &volumeProcessor{}
	processors[publishKey] = &volumeProcessor{}
	processors[unpublishKey] = &volumeProcessor{}
}

// volumeProcessor process volume
type volumeProcessor struct {
}

func (m *volumeProcessor) Process(message model.Message) {
	klog.Info("process volume started")
	back, err := beehiveContext.SendSync(modules.EdgedModuleName, message, constants.CSISyncMsgRespTimeout)
	klog.Infof("process volume get: req[%+v], back[%+v], err[%+v]", message, back, err)
	if err != nil {
		klog.Errorf("process volume send to edged failed: %v", err)
	}

	resp := message.NewRespByMessage(&message, back.GetContent())
	sendToCloud(resp)
	klog.Infof("process volume send to cloud resp[%+v]", resp)
}
