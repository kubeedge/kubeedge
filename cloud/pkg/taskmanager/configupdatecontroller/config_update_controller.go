/*
Copyright 2024 The KubeEdge Authors.

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

package configupdatecontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryType "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/api/apis/operations/v1alpha1"
	crdClientset "github.com/kubeedge/api/client/clientset/versioned"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util/controller"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util/manager"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

type ConfigUpdateController struct {
	sync.Mutex
	*controller.BaseController
}

var cache *manager.TaskCache

func NewConfigUpdateController(messageChan chan util.TaskMessage) (*ConfigUpdateController, error) {
	var err error
	cache, err = manager.NewTaskCache(
		informers.GetInformersManager().GetKubeEdgeInformerFactory().Operations().V1alpha1().ConfigUpdateJobs().Informer())
	if err != nil {
		klog.Warningf("Create ConfigUpdateJob manager failed with error: %s", err)
		return nil, err
	}
	return &ConfigUpdateController{
		BaseController: &controller.BaseController{
			Informer:    informers.GetInformersManager().GetKubeInformerFactory(),
			TaskManager: cache,
			MessageChan: messageChan,
			CrdClient:   client.GetCRDClient(),
			KubeClient:  keclient.GetKubeClient(),
		},
	}, nil
}

func (ndc *ConfigUpdateController) Start() error {
	go ndc.startSync()
	return nil
}

func (ndc *ConfigUpdateController) startSync() {
	configUpdateList, err := ndc.CrdClient.OperationsV1alpha1().ConfigUpdateJobs().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Errorf(err.Error())
		os.Exit(2)
	}
	for _, configUpdate := range configUpdateList.Items {
		if fsm.TaskFinish(configUpdate.Status.State) {
			continue
		}
		ndc.configUpdateJobAdded(&configUpdate)
	}
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop sync ConfigUpdateJob")
			return
		case e := <-ndc.TaskManager.Events():
			update, ok := e.Object.(*v1alpha1.ConfigUpdateJob)
			if !ok {
				klog.Warningf("object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				ndc.configUpdateJobAdded(update)
			case watch.Deleted:
				ndc.configUpdateJobDeleted(update)
			case watch.Modified:
				ndc.configUpdateJobUpdated(update)
			default:
				klog.Warningf("ConfigUpdateJob event type: %s unsupported", e.Type)
			}
		}
	}
}

// configUpdateJobAdded is used to process addition of new configUpdateJob in apiserver
func (ndc *ConfigUpdateController) configUpdateJobAdded(update *v1alpha1.ConfigUpdateJob) {
	klog.V(4).Infof("add ConfigUpdateJob: %v", update)
	// store in cache map
	ndc.TaskManager.CacheMap.Store(update.Name, update)

	// If config update job status is completed, we don't need to send update message
	if fsm.TaskFinish(update.Status.State) {
		klog.Warning("The configUpdateJob is completed, don't send update message again")
		return
	}

	ndc.processConfigUpdate(update)
}

// configUpdateJobDeleted is used to process deleted ConfigUpdateJob in apiserver
func (ndc *ConfigUpdateController) configUpdateJobDeleted(update *v1alpha1.ConfigUpdateJob) {
	// just need to delete from cache map
	ndc.TaskManager.CacheMap.Delete(update.Name)
	klog.Errorf("config update job %s delete", update.Name)
	ndc.MessageChan <- util.TaskMessage{
		Type:     util.TaskConfigUpdate,
		Name:     update.Name,
		ShutDown: true,
	}
}

// configUpdateJobUpdated is used to process updated ConfigUpdateJob in apiserver
func (ndc *ConfigUpdateController) configUpdateJobUpdated(update *v1alpha1.ConfigUpdateJob) {
	oldValue, ok := ndc.TaskManager.CacheMap.Load(update.Name)
	old := oldValue.(*v1alpha1.ConfigUpdateJob)
	if !ok {
		klog.Infof("Update %s not exist, and store it first", update.Name)
		// If ConfigUpdateJob not present in ConfigUpdate map means it is not modified and added.
		ndc.configUpdateJobAdded(update)
		return
	}

	// store in cache map
	ndc.TaskManager.CacheMap.Store(update.Name, update)

	node := checkUpdateNode(old, update)
	if node == nil {
		klog.Info("none node update")
		return
	}

	ndc.MessageChan <- util.TaskMessage{
		Type:   util.TaskConfigUpdate,
		Name:   update.Name,
		Status: *node,
	}
}

func (ndc *ConfigUpdateController) ReportNodeStatus(taskID, nodeID string, event fsm.Event) (api.State, error) {
	nodeFSM := NewConfigUpdateNodeFSM(taskID, nodeID)
	err := nodeFSM.AllowTransit(event)
	if err != nil {
		return "", err
	}
	state, err := nodeFSM.CurrentState()
	if err != nil {
		return "", err
	}
	ndc.Lock()
	defer ndc.Unlock()
	err = nodeFSM.Transit(event)
	if err != nil {
		return "", err
	}
	checkStatusChanged(nodeFSM, state)
	state, err = nodeFSM.CurrentState()
	if err != nil {
		return "", err
	}
	return state, nil
}

func (ndc *ConfigUpdateController) ReportTaskStatus(taskID string, event fsm.Event) (api.State, error) {
	taskFSM := NewConfigUpdateTaskFSM(taskID)
	state, err := taskFSM.CurrentState()
	if err != nil {
		return "", err
	}
	err = taskFSM.AllowTransit(event)
	if err != nil {
		return "", err
	}
	err = taskFSM.Transit(event)
	if err != nil {
		return "", err
	}
	checkStatusChanged(taskFSM, state)
	return taskFSM.CurrentState()
}

func checkStatusChanged(nodeFSM *fsm.FSM, state api.State) {
	err := wait.Poll(100*time.Millisecond, time.Second, func() (bool, error) {
		nowState, err := nodeFSM.CurrentState()
		if err != nil {
			return false, nil
		}
		if nowState == state {
			return false, nil
		}
		return true, err
	})
	if err != nil {
		klog.V(4).Infof("check status changed failed: %s", err.Error())
	}
}

func (ndc *ConfigUpdateController) StageCompleted(taskID string, state api.State) bool {
	taskFSM := NewConfigUpdateTaskFSM(taskID)
	return taskFSM.TaskStagCompleted(state)
}

func (ndc *ConfigUpdateController) GetNodeStatus(name string) ([]v1alpha1.TaskStatus, error) {
	configUpdate, err := ndc.CrdClient.OperationsV1alpha1().ConfigUpdateJobs().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return configUpdate.Status.Status, nil
}

func (ndc *ConfigUpdateController) UpdateNodeStatus(name string, nodeStatus []v1alpha1.TaskStatus) error {
	configUpdate, err := ndc.CrdClient.OperationsV1alpha1().ConfigUpdateJobs().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	status := configUpdate.Status
	status.Status = nodeStatus
	err = patchStatus(configUpdate, status, ndc.CrdClient)
	if err != nil {
		return err
	}
	return nil
}

// processConfigUpdate do the config update operation on node
func (ndc *ConfigUpdateController) processConfigUpdate(update *v1alpha1.ConfigUpdateJob) {
	updateMsg := commontypes.ConfigUpdateJobRequest{
		UpdateFields: update.Spec.UpdateFields,
		UpdateID:     update.Name,
	}

	tolerate, err := strconv.ParseFloat(update.Spec.FailureTolerate, 64)
	if err != nil {
		klog.Errorf("convert FailureTolerate to float64 failed: %v", err)
		tolerate = 0.1
	}

	concurrency := update.Spec.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	klog.V(4).Infof("deal task message: %v", update)
	ndc.MessageChan <- util.TaskMessage{
		Type:            util.TaskConfigUpdate,
		Name:            update.Name,
		TimeOutSeconds:  update.Spec.TimeoutSeconds,
		Concurrency:     concurrency,
		FailureTolerate: tolerate,
		NodeNames:       update.Spec.NodeNames,
		LabelSelector:   update.Spec.LabelSelector,
		Status:          v1alpha1.TaskStatus{},
		Msg:             updateMsg,
	}
}

func patchStatus(configUpdate *v1alpha1.ConfigUpdateJob, status v1alpha1.ConfigUpdateJobStatus, crdClient crdClientset.Interface) error {
	oldData, err := json.Marshal(configUpdate)
	if err != nil {
		return fmt.Errorf("failed to marshal the old ConfigUpdateJob(%s): %v", configUpdate.Name, err)
	}
	configUpdate.Status = status
	newData, err := json.Marshal(configUpdate)
	if err != nil {
		return fmt.Errorf("failed to marshal the new ConfigUpdateJob(%s): %v", configUpdate.Name, err)
	}

	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return fmt.Errorf("failed to create a merge patch: %v", err)
	}

	result, err := crdClient.OperationsV1alpha1().ConfigUpdateJobs().Patch(context.TODO(), configUpdate.Name, apimachineryType.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
	if err != nil {
		return fmt.Errorf("failed to patch ConfigUpdateJobs status: %v", err)
	}
	klog.V(4).Info("patch config update task status result: ", result)
	return nil
}

func checkUpdateNode(old, new *v1alpha1.ConfigUpdateJob) *v1alpha1.TaskStatus {
	if len(old.Status.Status) == 0 {
		return nil
	}
	for i, updateNode := range new.Status.Status {
		oldNode := old.Status.Status[i]
		if !util.NodeUpdated(oldNode, updateNode) {
			continue
		}
		return &updateNode
	}
	return nil
}
