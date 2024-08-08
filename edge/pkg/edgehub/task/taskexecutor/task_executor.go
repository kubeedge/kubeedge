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

package taskexecutor

import (
	"fmt"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

func init() {
	Register(TaskUpgrade, NewUpgradeExecutor())
	Register(TaskPrePull, NewPrePullExecutor())
}

type Executor interface {
	Name() string
	Do(types.NodeTaskRequest) (fsm.Event, error)
}

type BaseExecutor struct {
	name    string
	methods map[string]func(types.NodeTaskRequest) fsm.Event
}

func (be *BaseExecutor) Name() string {
	return be.name
}

func NewBaseExecutor(name string, methods map[string]func(types.NodeTaskRequest) fsm.Event) *BaseExecutor {
	return &BaseExecutor{
		name:    name,
		methods: methods,
	}
}

func (be *BaseExecutor) Do(taskReq types.NodeTaskRequest) (fsm.Event, error) {
	method, ok := be.methods[taskReq.State]
	if !ok {
		err := fmt.Errorf("method %s in executor %s is not implemented", taskReq.State, taskReq.Type)
		klog.Warning(err.Error())
		return fsm.Event{}, err
	}
	return method(taskReq), nil
}

var (
	executors     = make(map[string]Executor)
	CommonMethods = map[string]func(types.NodeTaskRequest) fsm.Event{
		string(v1alpha1.TaskChecking): preCheck,
		string(v1alpha1.TaskInit):     normalInit,
	}
)

func Register(name string, executor Executor) {
	if _, ok := executors[name]; ok {
		klog.Warningf("executor %s exists", name)
	}
	executors[name] = executor
}

func GetExecutor(name string) (Executor, error) {
	executor, ok := executors[name]
	if !ok {
		return nil, fmt.Errorf("executor %s is not registered", name)
	}
	return executor, nil
}

func emptyInit(_ types.NodeTaskRequest) (event fsm.Event) {
	return fsm.Event{
		Type:   "Init",
		Action: v1alpha1.ActionSuccess,
	}
}
