package taskexecutor

import (
	"fmt"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/apis/fsm/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

func init() {
	Register(TaskUpgrade, NewUpgradeExecutor())
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
		klog.Warning("executor %s exists ", name)
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
