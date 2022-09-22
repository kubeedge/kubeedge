/*
Copyright 2022 The KubeEdge Authors.

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

package status

import (
	"errors"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	statustest "k8s.io/kubernetes/pkg/kubelet/status/testing"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/podmanager"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

var (
	podStatusResource = "new/podstatus/foo"
)

func init() {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	metaManager := &common.ModuleInfo{
		ModuleName: modules.MetaManagerModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(metaManager)
	beehiveContext.AddModuleGroup(modules.MetaManagerModuleName, "meta")

	edgeHub := &common.ModuleInfo{
		ModuleName: "websocket",
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(edgeHub)
	beehiveContext.AddModuleGroup("websocket", modules.HubGroup)

	edged := &common.ModuleInfo{
		ModuleName: "edged",
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(edged)
}

// Generate new instance of test pod with the same initial value.
func getTestPod() *v1.Pod {
	return &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:       "12345678",
			Name:      "foo",
			Namespace: "new",
		},
	}
}

func getPodStatus(reason string) v1.PodStatus {
	return v1.PodStatus{
		Reason: reason,
	}
}

func newTestManager() *manager {
	podManager := podmanager.NewPodManager()
	podManager.AddPod(getTestPod())
	metaClient := client.New()
	return NewManager(&fake.Clientset{}, podManager, &statustest.FakePodDeletionSafetyProvider{}, metaClient).(*manager)
}

func TestUpdatePodStatusSucceed(t *testing.T) {
	manager := newTestManager()
	manager.Start()

	testPod := getTestPod()

	newReason := "test reason"
	manager.SetPodStatus(testPod, getPodStatus(newReason))

	msg, err := beehiveContext.Receive(modules.MetaManagerModuleName)
	if err != nil {
		t.Fatalf("receive message error: %v", err)
	}

	if msg.GetResource() != podStatusResource &&
		msg.GetOperation() != model.UpdateOperation {
		t.Fatalf("unexpected message: %v", msg)
	}

	// send response
	ackMessage := model.NewMessage(msg.GetID()).
		SetResourceOperation(podStatusResource, "response").
		FillBody(constants.MessageSuccessfulContent)
	beehiveContext.SendResp(*ackMessage)

	time.Sleep(2 * time.Second)

	_, exist := manager.apiStatusVersions[kubetypes.MirrorPodUID(testPod.GetUID())]
	if !exist {
		t.Fatalf("pod %s status should exist in apiStatusVersions", testPod.GetName())
	}
}

func TestUpdatePodStatusTimeout(t *testing.T) {
	manager := newTestManager()
	manager.Start()

	testPod := getTestPod()

	oldReason := "test timeout"
	manager.SetPodStatus(testPod, getPodStatus(oldReason))

	msg, err := beehiveContext.Receive(modules.MetaManagerModuleName)
	if err != nil {
		t.Fatalf("receive message error: %v", err)
	}

	if msg.GetResource() != podStatusResource &&
		msg.GetOperation() != model.UpdateOperation {
		t.Fatalf("unexpected message: %v", msg)
	}

	client.SetSyncPeriod(1 * time.Second)
	client.SetSyncMsgRespTimeout(1 * time.Second)

	// updatePosStatus timeout is 6s and sync interval time is 10s
	// so we set max wait time to 20s and then check the result
	waitTime := 20 * time.Second

	<-time.After(waitTime)
	_, exist := manager.apiStatusVersions[kubetypes.MirrorPodUID(testPod.GetUID())]
	if exist {
		t.Fatalf("pod %s status should not exist in apiStatusVersions", testPod.GetName())
	}
}

func TestUpdatePodStatusFailure(t *testing.T) {
	manager := newTestManager()
	manager.Start()

	testPod := getTestPod()

	oldReason := "update fail"
	manager.SetPodStatus(testPod, getPodStatus(oldReason))

	msg, err := beehiveContext.Receive(modules.MetaManagerModuleName)
	if err != nil {
		t.Fatalf("receive message error: %v", err)
	}

	if msg.GetResource() != podStatusResource &&
		msg.GetOperation() != model.UpdateOperation {
		t.Fatalf("unexpected message: %v", msg)
	}

	// send response
	err = errors.New("update failed")
	ackMessage := model.NewMessage(msg.GetID()).SetResourceOperation(podStatusResource, "response").FillBody(err)
	beehiveContext.SendResp(*ackMessage)

	_, exist := manager.apiStatusVersions[kubetypes.MirrorPodUID(testPod.GetUID())]
	if exist {
		t.Fatalf("pod %s status should not exist in apiStatusVersions", testPod.GetName())
	}
}
