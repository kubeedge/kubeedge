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

package manager

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/api/apis/operations/v1alpha1"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/nodeupgradecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util/controller"
	"github.com/kubeedge/kubeedge/common/constants"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

const TimeOutSecond = 300

type Executor struct {
	task           util.TaskMessage
	statusChan     chan *v1alpha1.TaskStatus
	nodes          []v1alpha1.TaskStatus
	controller     controller.Controller
	maxFailedNodes float64
	failedNodes    map[string]bool
	workers        workers
}

func NewExecutorMachine(messageChan chan util.TaskMessage, downStreamChan chan model.Message) (*ExecutorMachine, error) {
	executorMachine = &ExecutorMachine{
		kubeClient:     client.GetKubeClient(),
		executors:      map[string]*Executor{},
		messageChan:    messageChan,
		downStreamChan: downStreamChan,
	}
	return executorMachine, nil
}

func GetExecutorMachine() *ExecutorMachine {
	return executorMachine
}

// Start ExecutorMachine
func (em *ExecutorMachine) Start() error {
	klog.Info("Start ExecutorMachine")

	go em.syncTask()

	return nil
}

// syncTask is used to get events from informer
func (em *ExecutorMachine) syncTask() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop sync tasks")
			return
		case msg := <-em.messageChan:
			if msg.ShutDown {
				klog.Errorf("delete executor %s ", msg.Name)
				DeleteExecutor(msg)
				break
			}
			err := GetExecutor(msg).HandleMessage(msg.Status)
			if err != nil {
				klog.Errorf("Failed to handel %s message due to error %s", msg.Type, err.Error())
				break
			}
		}
	}
}

type ExecutorMachine struct {
	kubeClient     kubernetes.Interface
	executors      map[string]*Executor
	messageChan    chan util.TaskMessage
	downStreamChan chan model.Message
	sync.Mutex
}

var executorMachine *ExecutorMachine

func GetExecutor(msg util.TaskMessage) *Executor {
	executorMachine.Lock()
	e, ok := executorMachine.executors[fmt.Sprintf("%s::%s", msg.Type, msg.Name)]
	executorMachine.Unlock()
	if ok && e != nil {
		return e
	}
	e, err := initExecutor(msg)
	if err != nil {
		klog.Errorf("executor init failed, error: %s", err.Error())
		return nil
	}
	return e
}

func DeleteExecutor(msg util.TaskMessage) {
	executorMachine.Lock()
	defer executorMachine.Unlock()
	delete(executorMachine.executors, fmt.Sprintf("%s::%s", msg.Type, msg.Name))
}

func (e *Executor) HandleMessage(status v1alpha1.TaskStatus) error {
	if e == nil {
		return fmt.Errorf("executor is nil")
	}
	e.statusChan <- &status
	return nil
}

func (e *Executor) initMessage(node v1alpha1.TaskStatus) *model.Message {
	// delete it in 1.18
	if e.task.Type == util.TaskUpgrade {
		msg := e.initHistoryMessage(node)
		if msg != nil {
			klog.Warningf("send history message to node")
			return msg
		}
	}

	msg := model.NewMessage("")
	resource := buildTaskResource(e.task.Type, e.task.Name, node.NodeName)

	taskReq := commontypes.NodeTaskRequest{
		TaskID: e.task.Name,
		Type:   e.task.Type,
		State:  string(node.State),
	}
	taskReq.Item = e.task.Msg
	if node.State == api.TaskChecking {
		taskReq.Item = commontypes.NodePreCheckRequest{
			CheckItem: e.task.CheckItem,
		}
	}
	msg.BuildRouter(modules.TaskManagerModuleName, modules.TaskManagerModuleGroup, resource, e.task.Type).
		FillBody(taskReq)
	return msg
}

func (e *Executor) initHistoryMessage(node v1alpha1.TaskStatus) *model.Message {
	resource := buildUpgradeResource(e.task.Name, node.NodeName)
	req := e.task.Msg.(commontypes.NodeUpgradeJobRequest)
	upgradeController := e.controller.(*nodeupgradecontroller.NodeUpgradeController)
	edgeVersion, err := upgradeController.GetNodeVersion(node.NodeName)
	if err != nil {
		klog.Errorf("get node version failed: %s", err.Error())
		return nil
	}
	less, err := util.VersionLess(edgeVersion, "v1.16.0")
	if err != nil {
		klog.Errorf("version less failed: %s", err.Error())
		return nil
	}
	if !less {
		return nil
	}
	klog.Warningf("edge version is %s, is less than version %s", edgeVersion, "v1.16.0")
	upgradeReq := commontypes.NodeUpgradeJobRequest{
		UpgradeID:   e.task.Name,
		HistoryID:   uuid.New().String(),
		UpgradeTool: "keadm",
		Version:     req.Version,
		Image:       req.Image,
	}
	msg := model.NewMessage("")
	msg.BuildRouter(modules.NodeUpgradeJobControllerModuleName, modules.NodeUpgradeJobControllerModuleGroup, resource, util.TaskUpgrade).
		FillBody(upgradeReq)
	return msg
}

func initExecutor(message util.TaskMessage) (*Executor, error) {
	controller, err := controller.GetController(message.Type)
	if err != nil {
		return nil, err
	}
	nodeStatus, err := controller.GetNodeStatus(message.Name)
	if err != nil {
		return nil, err
	}
	if len(nodeStatus) == 0 {
		nodeList := controller.ValidateNode(message)
		if len(nodeList) == 0 {
			return nil, fmt.Errorf("no node need to be upgrade")
		}
		nodeStatus = make([]v1alpha1.TaskStatus, len(nodeList))
		for i, node := range nodeList {
			nodeStatus[i] = v1alpha1.TaskStatus{NodeName: node.Name}
		}
		err = controller.UpdateNodeStatus(message.Name, nodeStatus)
		if err != nil {
			return nil, err
		}
	}
	e := &Executor{
		task:           message,
		statusChan:     make(chan *v1alpha1.TaskStatus, 10),
		nodes:          nodeStatus,
		controller:     controller,
		maxFailedNodes: float64(len(nodeStatus)) * (message.FailureTolerate),
		failedNodes:    map[string]bool{},
		workers: workers{
			number:       int(message.Concurrency),
			jobs:         make(map[string]int),
			shuttingDown: false,
			Mutex:        sync.Mutex{},
		},
	}
	go e.start()
	executorMachine.executors[fmt.Sprintf("%s::%s", message.Type, message.Name)] = e
	return e, nil
}

func (e *Executor) start() {
	index, err := e.initWorker(0)
	if err != nil {
		klog.Error(err.Error())
		return
	}
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop sync tasks")
			return
		case status := <-e.statusChan:
			if reflect.DeepEqual(*status, v1alpha1.TaskStatus{}) {
				break
			}
			if !e.controller.StageCompleted(e.task.Name, status.State) {
				break
			}
			var endNode int
			endNode, err = e.workers.endJob(status.NodeName)
			if err != nil {
				klog.Error(err.Error())
				break
			}

			e.nodes[endNode] = *status
			err = e.dealFailedNode(*status)
			if err != nil {
				klog.Warning(err.Error())
				break
			}

			if index >= len(e.nodes) {
				if len(e.workers.jobs) != 0 {
					break
				}
				var state api.State
				state, err = e.completedTaskStage()
				if err != nil {
					klog.Error(err.Error())
					break
				}
				if fsm.TaskFinish(state) {
					DeleteExecutor(e.task)
					klog.Infof("task %s is finish", e.task.Name)
					return
				}

				// next stage
				index = 0
			}

			index, err = e.initWorker(index)
			if err != nil {
				klog.Error(err.Error())
			}
		}
	}
}

func (e *Executor) dealFailedNode(node v1alpha1.TaskStatus) error {
	if node.State == api.TaskFailed {
		e.failedNodes[node.NodeName] = true
	}
	if float64(len(e.failedNodes)) < e.maxFailedNodes {
		return nil
	}
	e.workers.shuttingDown = true
	if len(e.workers.jobs) > 0 {
		klog.Warningf("wait for all workers(%d/%d) for task %s to finish running ", len(e.workers.jobs), e.workers.number, e.task.Name)
		return nil
	}

	errMsg := fmt.Sprintf("the number of failed nodes is %d/%d, which exceeds the failure tolerance threshold.", len(e.failedNodes), len(e.nodes))
	_, err := e.controller.ReportTaskStatus(e.task.Name, fsm.Event{
		Type:   node.Event,
		Action: api.ActionFailure,
		Msg:    errMsg,
	})
	if err != nil {
		return fmt.Errorf("%s, report status failed, %s", errMsg, err.Error())
	}
	return errors.New(errMsg)
}

func (e *Executor) completedTaskStage() (api.State, error) {
	var event = e.nodes[0].Event
	for _, node := range e.nodes {
		if node.State != api.TaskFailed {
			event = node.Event
			break
		}
	}
	state, err := e.controller.ReportTaskStatus(e.task.Name, fsm.Event{
		Type:   event,
		Action: api.ActionSuccess,
	})
	if err != nil {
		return "", err
	}
	return state, nil
}

func (e *Executor) initWorker(index int) (int, error) {
	for ; index < len(e.nodes); index++ {
		node := e.nodes[index]
		if e.controller.StageCompleted(e.task.Name, node.State) {
			err := e.dealFailedNode(node)
			if err != nil {
				return 0, err
			}
			continue
		}
		err := e.workers.addJob(node, index, e)
		if err != nil {
			klog.V(4).Info(err.Error())
			break
		}
	}
	return index, nil
}

type workers struct {
	number int
	jobs   map[string]int
	sync.Mutex
	shuttingDown bool
}

func (w *workers) addJob(node v1alpha1.TaskStatus, index int, e *Executor) error {
	if w.shuttingDown {
		return fmt.Errorf("workers is stopped")
	}
	w.Lock()
	if len(w.jobs) >= w.number {
		w.Unlock()
		return fmt.Errorf("workers are all running, %v/%v", len(w.jobs), w.number)
	}
	w.jobs[node.NodeName] = index
	w.Unlock()
	msg := e.initMessage(node)
	go e.handelTimeOutJob(index)
	executorMachine.downStreamChan <- *msg
	return nil
}

func (e *Executor) handelTimeOutJob(index int) {
	lastState := e.nodes[index].State
	timeoutSecond := *e.task.TimeOutSeconds
	if timeoutSecond == 0 {
		timeoutSecond = TimeOutSecond
	}
	err := wait.Poll(1*time.Second, time.Duration(timeoutSecond)*time.Second, func() (bool, error) {
		if lastState != e.nodes[index].State || fsm.TaskFinish(e.nodes[index].State) {
			return true, nil
		}
		klog.V(4).Infof("node %s stage is not completed", e.nodes[index].NodeName)
		return false, nil
	})
	if err != nil {
		_, err = e.controller.ReportNodeStatus(e.task.Name, e.nodes[index].NodeName, fsm.Event{
			Type:   api.EventTimeOut,
			Action: api.ActionFailure,
			Msg:    fmt.Sprintf("node task %s execution timeout, %s", lastState, err.Error()),
		})
		if err != nil {
			klog.Warning(err.Error())
		}
	}
}

func (w *workers) endJob(job string) (int, error) {
	index, ok := w.jobs[job]
	if !ok {
		return index, fmt.Errorf("end job %s error, job not exist", job)
	}
	w.Lock()
	delete(w.jobs, job)
	w.Unlock()
	return index, nil
}

func buildTaskResource(task, taskID, nodeID string) string {
	resource := strings.Join([]string{task, taskID, "node", nodeID}, constants.ResourceSep)
	return resource
}

func buildUpgradeResource(upgradeID, nodeID string) string {
	resource := strings.Join([]string{util.TaskUpgrade, upgradeID, "node", nodeID}, constants.ResourceSep)
	return resource
}
