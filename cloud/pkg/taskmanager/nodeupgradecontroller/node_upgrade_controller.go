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

package nodeupgradecontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryType "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"

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

const NodeUpgrade = "NodeUpgradeController"

type NodeUpgradeController struct {
	sync.Mutex
	*controller.BaseController
}

var cache *manager.TaskCache

func NewNodeUpgradeController(messageChan chan util.TaskMessage) (*NodeUpgradeController, error) {
	var err error
	cache, err = manager.NewTaskCache(
		informers.GetInformersManager().GetKubeEdgeInformerFactory().Operations().V1alpha1().NodeUpgradeJobs().Informer())
	if err != nil {
		klog.Warningf("Create NodeUpgradeJob manager failed with error: %s", err)
		return nil, err
	}
	return &NodeUpgradeController{
		BaseController: &controller.BaseController{
			Informer:    informers.GetInformersManager().GetKubeInformerFactory(),
			TaskManager: cache,
			MessageChan: messageChan,
			CrdClient:   client.GetCRDClient(),
			KubeClient:  keclient.GetKubeClient(),
		},
	}, nil
}

func (ndc *NodeUpgradeController) ReportNodeStatus(taskID, nodeID string, event fsm.Event) (api.State, error) {
	nodeFSM := NewUpgradeNodeFSM(taskID, nodeID)
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

func (ndc *NodeUpgradeController) ReportTaskStatus(taskID string, event fsm.Event) (api.State, error) {
	taskFSM := NewUpgradeTaskFSM(taskID)
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

func (ndc *NodeUpgradeController) ValidateNode(taskMessage util.TaskMessage) []v1.Node {
	var validateNodes []v1.Node
	nodes := ndc.BaseController.ValidateNode(taskMessage)
	req, ok := taskMessage.Msg.(commontypes.NodeUpgradeJobRequest)
	if !ok {
		klog.Errorf("convert message to commontypes.NodeUpgradeJobRequest failed")
		return nil
	}
	for _, node := range nodes {
		if needUpgrade(node, req.Version) {
			validateNodes = append(validateNodes, node)
		}
	}
	return validateNodes
}

func (ndc *NodeUpgradeController) StageCompleted(taskID string, state api.State) bool {
	taskFSM := NewUpgradeTaskFSM(taskID)
	return taskFSM.TaskStagCompleted(state)
}

func (ndc *NodeUpgradeController) GetNodeStatus(name string) ([]v1alpha1.TaskStatus, error) {
	nodeUpgrade, err := ndc.CrdClient.OperationsV1alpha1().NodeUpgradeJobs().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return nodeUpgrade.Status.Status, nil
}

func (ndc *NodeUpgradeController) GetNodeVersion(name string) (string, error) {
	node, err := ndc.Informer.Core().V1().Nodes().Lister().Get(name)
	if err != nil {
		return "", err
	}
	strs := strings.Split(node.Status.NodeInfo.KubeletVersion, "-")
	return strs[2], nil
}

func (ndc *NodeUpgradeController) UpdateNodeStatus(name string, nodeStatus []v1alpha1.TaskStatus) error {
	nodeUpgrade, err := ndc.CrdClient.OperationsV1alpha1().NodeUpgradeJobs().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	status := nodeUpgrade.Status
	status.Status = nodeStatus
	err = patchStatus(nodeUpgrade, status, ndc.CrdClient)
	if err != nil {
		return err
	}
	return nil
}

func patchStatus(nodeUpgrade *v1alpha1.NodeUpgradeJob, status v1alpha1.NodeUpgradeJobStatus, crdClient crdClientset.Interface) error {
	oldData, err := json.Marshal(nodeUpgrade)
	if err != nil {
		return fmt.Errorf("failed to marshal the old NodeUpgradeJob(%s): %v", nodeUpgrade.Name, err)
	}
	nodeUpgrade.Status = status
	newData, err := json.Marshal(nodeUpgrade)
	if err != nil {
		return fmt.Errorf("failed to marshal the new NodeUpgradeJob(%s): %v", nodeUpgrade.Name, err)
	}

	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return fmt.Errorf("failed to create a merge patch: %v", err)
	}

	result, err := crdClient.OperationsV1alpha1().NodeUpgradeJobs().Patch(context.TODO(), nodeUpgrade.Name, apimachineryType.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
	if err != nil {
		return fmt.Errorf("failed to patch update NodeUpgradeJob status: %v", err)
	}
	klog.V(4).Info("patch upgrade task status result: ", result)
	return nil
}

func (ndc *NodeUpgradeController) Start() error {
	go ndc.startSync()
	return nil
}

func (ndc *NodeUpgradeController) startSync() {
	nodeUpgradeList, err := ndc.CrdClient.OperationsV1alpha1().NodeUpgradeJobs().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Error(err.Error())
		os.Exit(2)
	}
	for _, nodeUpgrade := range nodeUpgradeList.Items {
		if fsm.TaskFinish(nodeUpgrade.Status.State) {
			continue
		}
		ndc.nodeUpgradeJobAdded(&nodeUpgrade)
	}
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop sync NodeUpgradeJob")
			return
		case e := <-ndc.TaskManager.Events():
			upgrade, ok := e.Object.(*v1alpha1.NodeUpgradeJob)
			if !ok {
				klog.Warningf("object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				ndc.nodeUpgradeJobAdded(upgrade)
			case watch.Deleted:
				ndc.nodeUpgradeJobDeleted(upgrade)
			case watch.Modified:
				ndc.nodeUpgradeJobUpdated(upgrade)
			default:
				klog.Warningf("NodeUpgradeJob event type: %s unsupported", e.Type)
			}
		}
	}
}

// nodeUpgradeJobAdded is used to process addition of new NodeUpgradeJob in apiserver
func (ndc *NodeUpgradeController) nodeUpgradeJobAdded(upgrade *v1alpha1.NodeUpgradeJob) {
	klog.V(4).Infof("add NodeUpgradeJob: %v", upgrade)
	// store in cache map
	ndc.TaskManager.CacheMap.Store(upgrade.Name, upgrade)

	// If all or partial edge nodes upgrade is upgrading or completed, we don't need to send upgrade message
	if fsm.TaskFinish(upgrade.Status.State) {
		klog.Warning("The nodeUpgradeJob is completed, don't send upgrade message again")
		return
	}

	ndc.processUpgrade(upgrade)
}

// processUpgrade do the upgrade operation on node
func (ndc *NodeUpgradeController) processUpgrade(upgrade *v1alpha1.NodeUpgradeJob) {
	// if users specify Image, we'll use upgrade Version as its image tag, even though Image contains tag.
	// if not, we'll use default image: kubeedge/installation-package:${Version}
	var repo string
	var err error
	repo = "kubeedge/installation-package"
	if upgrade.Spec.Image != "" {
		repo, err = util.GetImageRepo(upgrade.Spec.Image)
		if err != nil {
			klog.Errorf("Image format is not right: %v", err)
			return
		}
	}
	imageTag := upgrade.Spec.Version
	image := fmt.Sprintf("%s:%s", repo, imageTag)

	var imageDigest string
	if g := upgrade.Spec.ImageDigestGatter; g != nil {
		switch {
		case g.Value != nil && *g.Value != "":
			imageDigest = *g.Value
		case g.RegistryAPI != nil:
			imageURL := fmt.Sprintf("%s/%s", g.RegistryAPI.Host, image)
			imageDigest, _ = getImageDigest(imageURL, g.RegistryAPI.Token)
		}
	}

	upgradeReq := commontypes.NodeUpgradeJobRequest{
		UpgradeID:           upgrade.Name,
		HistoryID:           uuid.New().String(),
		Version:             upgrade.Spec.Version,
		Image:               image,
		ImageDigest:         imageDigest,
		RequireConfirmation: upgrade.Spec.RequireConfirmation,
	}

	tolerate, err := strconv.ParseFloat(upgrade.Spec.FailureTolerate, 64)
	if err != nil {
		klog.Errorf("convert FailureTolerate to float64 failed: %v", err)
		tolerate = 0.1
	}

	concurrency := upgrade.Spec.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	klog.V(4).Infof("deal task message: %v", upgrade)
	ndc.MessageChan <- util.TaskMessage{
		Type:            util.TaskUpgrade,
		CheckItem:       upgrade.Spec.CheckItems,
		Name:            upgrade.Name,
		TimeOutSeconds:  upgrade.Spec.TimeoutSeconds,
		Concurrency:     concurrency,
		FailureTolerate: tolerate,
		NodeNames:       upgrade.Spec.NodeNames,
		LabelSelector:   upgrade.Spec.LabelSelector,
		Status:          v1alpha1.TaskStatus{},
		Msg:             upgradeReq,
	}
}

// GetImageDigest retrieves the digest of a given image from a registry
func getImageDigest(imageURL string, token string) (string, error) {
	// Parse the image reference (e.g., "docker.io/library/ubuntu:latest")
	ref, err := registry.ParseReference(imageURL)
	if err != nil {
		return "", err
	}
	// Create a new remote repository instance
	repository, err := remote.NewRepository(ref.Repository)
	if err != nil {
		return "", err
	}

	// If a token is provided, set up the authentication client
	if token != "" {
		credential := &auth.Credential{
			AccessToken: token,
		}
		repository.Client = &auth.Client{
			Client: retry.DefaultClient,
			Header: http.Header{
				"User-Agent": {"oras-go"},
			},
			Credential: func(ctx context.Context, host string) (auth.Credential, error) {
				return *credential, nil
			},
			Cache:              nil,
			ClientID:           "oras-client",
			ForceAttemptOAuth2: false,
		}
	}
	// Set up the context for the request
	ctx := context.Background()
	// Resolve the image reference to get the manifest descriptor
	descriptor, err := repository.Resolve(ctx, ref.Reference)
	if err != nil {
		return "", err
	}

	// Return the image digest
	return descriptor.Digest.String(), nil
}

func needUpgrade(node v1.Node, upgradeVersion string) bool {
	if util.FilterVersion(node.Status.NodeInfo.KubeletVersion, upgradeVersion) {
		klog.Warningf("Node(%s) version(%s) already on the expected version %s.", node.Name, node.Status.NodeInfo.KubeletVersion, upgradeVersion)
		return false
	}

	// if node is in Upgrading state, don't need upgrade
	if _, ok := node.Labels[util.NodeUpgradeJobStatusKey]; ok {
		klog.Warningf("Node(%s) is in upgrade state", node.Name)
		return false
	}

	return true
}

// nodeUpgradeJobDeleted is used to process deleted NodeUpgradeJob in apiserver
func (ndc *NodeUpgradeController) nodeUpgradeJobDeleted(upgrade *v1alpha1.NodeUpgradeJob) {
	// just need to delete from cache map
	ndc.TaskManager.CacheMap.Delete(upgrade.Name)
	klog.Errorf("upgrade job %s delete", upgrade.Name)
	ndc.MessageChan <- util.TaskMessage{
		Type:     util.TaskUpgrade,
		Name:     upgrade.Name,
		ShutDown: true,
	}
}

// upgradeAdded is used to process update of new NodeUpgradeJob in apiserver
func (ndc *NodeUpgradeController) nodeUpgradeJobUpdated(upgrade *v1alpha1.NodeUpgradeJob) {
	oldValue, ok := ndc.TaskManager.CacheMap.Load(upgrade.Name)
	old := oldValue.(*v1alpha1.NodeUpgradeJob)
	if !ok {
		klog.Infof("Upgrade %s not exist, and store it first", upgrade.Name)
		// If Upgrade not present in Upgrade map means it is not modified and added.
		ndc.nodeUpgradeJobAdded(upgrade)
		return
	}

	// store in cache map
	ndc.TaskManager.CacheMap.Store(upgrade.Name, upgrade)

	node := checkUpdateNode(old, upgrade)
	if node == nil {
		klog.Info("none node update")
		return
	}

	ndc.MessageChan <- util.TaskMessage{
		Type:   util.TaskUpgrade,
		Name:   upgrade.Name,
		Status: *node,
	}
}

func checkUpdateNode(old, new *v1alpha1.NodeUpgradeJob) *v1alpha1.TaskStatus {
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
