package processor

import (
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	cloudmodules "github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
)

// edgedInsertProcessor process add metadata
type edgedInsertProcessor struct {
	insertProcessor
}

func (p *edgedInsertProcessor) Process(message model.Message) {
	if err := p.insertProcessor.Process(message); err != nil {
		klog.Errorf("edged process insert failed: %v", err)
		return
	}

	imitator.DefaultV2Client.Inject(message)
	// Notify edged
	sendToEdged(&message, false)

	resp := message.NewRespByMessage(&message, OK)
	sendToCloud(resp)
}

// edgedUpdateProcessor process update metadata
type edgedUpdateProcessor struct {
	updateProcessor
}

func (p *edgedUpdateProcessor) Process(message model.Message) {
	if err := p.updateProcessor.Process(message); err != nil {
		klog.Errorf("edged process update failed: %v", err)
		return
	}

	imitator.DefaultV2Client.Inject(message)

	msgSource := message.GetSource()
	switch msgSource {
	case modules.EdgedModuleName:
		sendToCloud(&message)
		resp := message.NewRespByMessage(&message, OK)
		sendToEdged(resp, message.IsSync())
	case cloudmodules.EdgeControllerModuleName, cloudmodules.DynamicControllerModuleName:
		sendToEdged(&message, message.IsSync())
		resp := message.NewRespByMessage(&message, OK)
		sendToCloud(resp)
	case CloudFunctionModel:
		beehiveContext.Send(EdgeFunctionModel, message)
	case EdgeFunctionModel:
		sendToCloud(&message)
	default:
		klog.Errorf("unsupport message source, %s", msgSource)
	}
}

// edgedDeleteProcessor process delete metadata
type edgedDeleteProcessor struct {
	deleteProcessor
}

func (p *edgedDeleteProcessor) Process(message model.Message) {
	if err := p.deleteProcessor.Process(message); err != nil {
		klog.Errorf("edged process delete failed: %v", err)
		return
	}

	imitator.DefaultV2Client.Inject(message)

	_, resType, _ := parseResource(message.GetResource())
	if resType == model.ResourceTypePod && message.GetSource() == modules.EdgedModuleName {
		sendToCloud(&message)
		return
	}

	// Notify edged
	sendToEdged(&message, false)
	resp := message.NewRespByMessage(&message, OK)
	sendToCloud(resp)
}

// edgedQueryProcessor process query metadata
type edgedQueryProcessor struct {
	queryProcessor
}

func (p *edgedQueryProcessor) Process(message model.Message) {
	p.queryProcessor.Process(message)
}

// edgedResponseProcessor process response metadata
type edgedResponseProcessor struct {
	responseProcessor
}

func (p *edgedResponseProcessor) Process(message model.Message) {
	if err := p.responseProcessor.Process(message); err != nil {
		klog.Errorf("edged process response failed: %v", err)
		return
	}
	// Notify edged if the data is coming from cloud
	if message.GetSource() == CloudControlerModel {
		sendToEdged(&message, message.IsSync())
	} else {
		// Send to cloud if the update request is coming from edged
		sendToCloud(&message)
	}
}

func init() {
	var table = [...][2]interface{}{
		{model.InsertOperation, &edgedInsertProcessor{}},
		{model.DeleteOperation, &edgedDeleteProcessor{}},
		{model.UpdateOperation, &edgedUpdateProcessor{}},
		{model.QueryOperation, &edgedQueryProcessor{}},
		{model.ResponseOperation, &edgedResponseProcessor{}},
	}
	for _, row := range table {
		key := queryKey{
			operation: row[0].(string),
		}
		processors[key] = row[1].(Processor)
	}
}
