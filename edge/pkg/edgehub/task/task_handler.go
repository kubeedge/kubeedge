package task

import (
	"encoding/json"
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/msghandler"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/task/taskexecutor"
)

func init() {
	handler := &taskHandler{}
	msghandler.RegisterHandler(handler)
}

type taskHandler struct{}

func (th *taskHandler) Filter(message *model.Message) bool {
	name := message.GetGroup()
	return name == modules.TaskManagerModuleName
}

func (th *taskHandler) Process(message *model.Message, clientHub clients.Adapter) error {
	taskReq := &commontypes.NodeTaskRequest{}
	data, err := message.GetContentData()
	if err != nil {
		return fmt.Errorf("failed to get content data: %v", err)
	}
	err = json.Unmarshal(data, taskReq)
	if err != nil {
		return fmt.Errorf("unmarshal failed: %v", err)
	}
	executor, err := taskexecutor.GetExecutor(taskReq.Type)
	if err != nil {
		return err
	}
	event, err := executor.Do(*taskReq)
	if err != nil {
		return err
	}

	// use external tool like keadm
	if event.Action == "" {
		return nil
	}
	err = util.ReportUpgradeResult(options.GetEdgeCoreConfig(), taskReq.Type, taskReq.TaskID, event)
	if err != nil {
		return err
	}
	return nil
}
