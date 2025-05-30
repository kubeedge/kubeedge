/*
Copyright 2025 The KubeEdge Authors.

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

package taskmanager

import (
	"context"
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/taskmanager/actions"
	nodetaskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func InitRunner() {
	actions.Init()
}

// RunTask parses the message and runs the node task actions.
func RunTask(msg *model.Message) error {
	msgres := nodetaskmsg.ParseResource(msg.GetResource())
	runner := actions.GetRunner(msgres.ResourceType)
	if runner == nil {
		return fmt.Errorf("invalid resource type %s", msgres.ResourceType)
	}
	data, err := msg.GetContentData()
	if err != nil {
		return fmt.Errorf("failed to get node job message content data: %v", err)
	}
	runner.RunAction(context.Background(), msgres.JobName, msgres.NodeName, msg.GetOperation(), data)
	return nil
}
