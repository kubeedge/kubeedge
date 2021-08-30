package processor

import (
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
)

// functionActionProcessor process add "action" metadata
type functionActionProcessor struct {
	insertProcessor
}

func (m *functionActionProcessor) Process(message model.Message) {
	if err := m.insertProcessor.Process(message); err != nil {
		klog.Errorf("function process action failed: %v", err)
		return
	}

	beehiveContext.Send(EdgeFunctionModel, message)
}

// functionActionResultProcessor process "action_result" metadata
type functionActionResultProcessor struct {
	insertProcessor
}

func (m *functionActionResultProcessor) Process(message model.Message) {
	if err := m.insertProcessor.Process(message); err != nil {
		klog.Errorf("function process query failed: %v", err)
		return
	}

	sendToCloud(&message)
}

func init() {
	ncKey := queryKey{
		operation: OperationFunctionAction,
	}

	qKey := queryKey{
		operation: OperationFunctionActionResult,
	}

	processors[ncKey] = &functionActionProcessor{}
	processors[qKey] = &functionActionResultProcessor{}
}
