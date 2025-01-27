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

package imageprepullcontroller

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

type ImagePrePullController struct {
	sync.Mutex
	*controller.BaseController
}

var cache *manager.TaskCache

func NewImagePrePullController(messageChan chan util.TaskMessage) (*ImagePrePullController, error) {
	var err error
	cache, err = manager.NewTaskCache(
		informers.GetInformersManager().GetKubeEdgeInformerFactory().Operations().V1alpha1().ImagePrePullJobs().Informer())
	if err != nil {
		klog.Warningf("Create image pre pull controller failed with error: %s", err)
		return nil, err
	}
	return &ImagePrePullController{
		BaseController: &controller.BaseController{
			Informer:    informers.GetInformersManager().GetKubeInformerFactory(),
			TaskManager: cache,
			MessageChan: messageChan,
			CrdClient:   client.GetCRDClient(),
			KubeClient:  keclient.GetKubeClient(),
		},
	}, nil
}

func (ndc *ImagePrePullController) ReportNodeStatus(taskID, nodeID string, event fsm.Event) (api.State, error) {
	nodeFSM := NewImagePrePullNodeFSM(taskID, nodeID)
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

func (ndc *ImagePrePullController) ReportTaskStatus(taskID string, event fsm.Event) (api.State, error) {
	taskFSM := NewImagePrePullTaskFSM(taskID)
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

func (ndc *ImagePrePullController) StageCompleted(taskID string, state api.State) bool {
	taskFSM := NewImagePrePullTaskFSM(taskID)
	return taskFSM.TaskStagCompleted(state)
}

func (ndc *ImagePrePullController) GetNodeStatus(name string) ([]v1alpha1.TaskStatus, error) {
	imagePrePull, err := ndc.CrdClient.OperationsV1alpha1().ImagePrePullJobs().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	statusList := make([]v1alpha1.TaskStatus, len(imagePrePull.Status.Status))
	for i, status := range imagePrePull.Status.Status {
		if status.TaskStatus == nil {
			statusList[i] = v1alpha1.TaskStatus{}
			continue
		}
		statusList[i] = *status.TaskStatus
	}
	return statusList, nil
}

func (ndc *ImagePrePullController) UpdateNodeStatus(name string, nodeStatus []v1alpha1.TaskStatus) error {
	imagePrePull, err := ndc.CrdClient.OperationsV1alpha1().ImagePrePullJobs().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	status := imagePrePull.Status
	statusList := make([]v1alpha1.ImagePrePullStatus, len(nodeStatus))
	for i := 0; i < len(nodeStatus); i++ {
		statusList[i].TaskStatus = &nodeStatus[i]
	}
	status.Status = statusList
	err = patchStatus(imagePrePull, status, ndc.CrdClient)
	if err != nil {
		return err
	}
	return nil
}

func patchStatus(imagePrePullJob *v1alpha1.ImagePrePullJob, status v1alpha1.ImagePrePullJobStatus, crdClient crdClientset.Interface) error {
	oldData, err := json.Marshal(imagePrePullJob)
	if err != nil {
		return fmt.Errorf("failed to marshal the old ImagePrePullJob(%s): %v", imagePrePullJob.Name, err)
	}
	imagePrePullJob.Status = status
	newData, err := json.Marshal(imagePrePullJob)
	if err != nil {
		return fmt.Errorf("failed to marshal the new ImagePrePullJob(%s): %v", imagePrePullJob.Name, err)
	}

	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return fmt.Errorf("failed to create a merge patch: %v", err)
	}

	result, err := crdClient.OperationsV1alpha1().ImagePrePullJobs().Patch(context.TODO(), imagePrePullJob.Name, apimachineryType.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
	if err != nil {
		return fmt.Errorf("failed to patch update ImagePrePullJob status: %v", err)
	}
	klog.V(4).Info("patch update task status result: ", result)
	return nil
}

func (ndc *ImagePrePullController) Start() error {
	go ndc.startSync()
	return nil
}

func (ndc *ImagePrePullController) startSync() {
	imagePrePullList, err := ndc.CrdClient.OperationsV1alpha1().ImagePrePullJobs().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Error(err.Error())
		os.Exit(2)
	}
	for _, imagePrePull := range imagePrePullList.Items {
		if fsm.TaskFinish(imagePrePull.Status.State) {
			continue
		}
		ndc.imagePrePullJobAdded(&imagePrePull)
	}
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop sync ImagePrePullJob")
			return
		case e := <-ndc.TaskManager.Events():
			prePull, ok := e.Object.(*v1alpha1.ImagePrePullJob)
			if !ok {
				klog.Warningf("object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				ndc.imagePrePullJobAdded(prePull)
			case watch.Deleted:
				ndc.imagePrePullJobDeleted(prePull)
			case watch.Modified:
				ndc.imagePrePullJobUpdated(prePull)
			default:
				klog.Warningf("ImagePrePullJob event type: %s unsupported", e.Type)
			}
		}
	}
}

// imagePrePullJobAdded is used to process addition of new ImagePrePullJob in apiserver
func (ndc *ImagePrePullController) imagePrePullJobAdded(imagePrePull *v1alpha1.ImagePrePullJob) {
	klog.V(4).Infof("add ImagePrePullJob: %v", imagePrePull)
	// store in cache map
	ndc.TaskManager.CacheMap.Store(imagePrePull.Name, imagePrePull)

	// If all or partial edge nodes image pull is pulling or completed, we don't need to send pull message
	if fsm.TaskFinish(imagePrePull.Status.State) {
		klog.Warning("The ImagePrePullJob is completed, don't send pull message again")
		return
	}

	ndc.processPrePull(imagePrePull)
}

// processPrePull do the pre pull operation on node
func (ndc *ImagePrePullController) processPrePull(imagePrePull *v1alpha1.ImagePrePullJob) {
	imagePrePullTemplateInfo := imagePrePull.Spec.ImagePrePullTemplate
	imagePrePullRequest := commontypes.ImagePrePullJobRequest{
		Images:     imagePrePullTemplateInfo.Images,
		Secret:     imagePrePullTemplateInfo.ImageSecret,
		RetryTimes: imagePrePullTemplateInfo.RetryTimes,
		CheckItems: imagePrePullTemplateInfo.CheckItems,
	}
	tolerate, err := strconv.ParseFloat(imagePrePull.Spec.ImagePrePullTemplate.FailureTolerate, 64)
	if err != nil {
		klog.Errorf("convert FailureTolerate to float64 failed: %v", err)
		tolerate = 0.1
	}

	concurrency := imagePrePull.Spec.ImagePrePullTemplate.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	klog.V(4).Infof("deal task message: %v", imagePrePull)
	ndc.MessageChan <- util.TaskMessage{
		Type:            util.TaskPrePull,
		CheckItem:       imagePrePull.Spec.ImagePrePullTemplate.CheckItems,
		Name:            imagePrePull.Name,
		TimeOutSeconds:  imagePrePull.Spec.ImagePrePullTemplate.TimeoutSeconds,
		Concurrency:     concurrency,
		FailureTolerate: tolerate,
		NodeNames:       imagePrePull.Spec.ImagePrePullTemplate.NodeNames,
		LabelSelector:   imagePrePull.Spec.ImagePrePullTemplate.LabelSelector,
		Status:          v1alpha1.TaskStatus{},
		Msg:             imagePrePullRequest,
	}
}

// imagePrePullJobDeleted is used to process deleted ImagePrePullJob in apiserver
func (ndc *ImagePrePullController) imagePrePullJobDeleted(imagePrePull *v1alpha1.ImagePrePullJob) {
	// just need to delete from cache map
	ndc.TaskManager.CacheMap.Delete(imagePrePull.Name)
	klog.Errorf("image pre pull job %s delete", imagePrePull.Name)
	ndc.MessageChan <- util.TaskMessage{
		Type:     util.TaskPrePull,
		Name:     imagePrePull.Name,
		ShutDown: true,
	}
}

// imagePrePullJobUpdated is used to process update of new ImagePrePullJob in apiserver
func (ndc *ImagePrePullController) imagePrePullJobUpdated(pullJob *v1alpha1.ImagePrePullJob) {
	oldValue, ok := ndc.TaskManager.CacheMap.Load(pullJob.Name)
	old := oldValue.(*v1alpha1.ImagePrePullJob)
	if !ok {
		klog.Infof("Update %s not exist, and store it first", pullJob.Name)
		// If PrePull not present in PrePull map means it is not modified and added.
		ndc.imagePrePullJobAdded(pullJob)
		return
	}

	// store in cache map
	ndc.TaskManager.CacheMap.Store(pullJob.Name, pullJob)

	node := checkUpdateNode(old, pullJob)
	if node == nil {
		klog.Info("none node update")
		return
	}

	ndc.MessageChan <- util.TaskMessage{
		Type:   util.TaskPrePull,
		Name:   pullJob.Name,
		Status: *node,
	}
}

func checkUpdateNode(old, new *v1alpha1.ImagePrePullJob) *v1alpha1.TaskStatus {
	if len(old.Status.Status) == 0 {
		return nil
	}
	for i, updateNode := range new.Status.Status {
		oldNode := old.Status.Status[i]
		if !util.NodeUpdated(*oldNode.TaskStatus, *updateNode.TaskStatus) {
			continue
		}
		return updateNode.TaskStatus
	}
	return nil
}
