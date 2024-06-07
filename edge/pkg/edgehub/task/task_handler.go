/*
Copyright 2023 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

func (th *taskHandler) Process(message *model.Message, _ clients.Adapter) error {
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
	//Or the task needs to use the goroutine to control the message reply itself.
	if event.Action == "" {
		return nil
	}

	resp := commontypes.NodeTaskResponse{
		NodeName: options.GetEdgeCoreConfig().Modules.Edged.HostnameOverride,
		Event:    event.Type,
		Action:   event.Action,
		Reason:   event.Msg,
	}
	util.ReportTaskResult(taskReq.Type, taskReq.TaskID, resp)
	return nil
}
