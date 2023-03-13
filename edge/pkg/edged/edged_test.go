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

package edged

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletConfig "k8s.io/kubernetes/pkg/kubelet/config"

	"github.com/kubeedge/beehive/pkg/common"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

func init() {
	cfg := v1alpha2.NewDefaultEdgeCoreConfig()
	config.InitConfigure(cfg.Modules.Edged)

	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	edged := &common.ModuleInfo{
		ModuleName: modules.EdgedModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(edged)
	beehiveContext.AddModuleGroup(modules.EdgedModuleName, modules.EdgedGroup)

	meta := &common.ModuleInfo{
		ModuleName: modules.MetaManagerModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(meta)
	beehiveContext.AddModuleGroup(modules.MetaManagerModuleName, modules.MetaGroup)
}

func TestRegister(t *testing.T) {
	defaultTailedKubeletConfig := v1alpha2.TailoredKubeletConfiguration{}
	v1alpha2.SetDefaultsKubeletConfiguration(&defaultTailedKubeletConfig)

	tests := []struct {
		name  string
		edged *v1alpha2.Edged
		want  *v1alpha2.Edged
	}{
		{
			name: "Register Edged Succeed",
			edged: &v1alpha2.Edged{
				Enable: true,
				TailoredKubeletFlag: v1alpha2.TailoredKubeletFlag{
					HostnameOverride: "testnode2",
					ContainerRuntimeOptions: v1alpha2.ContainerRuntimeOptions{
						ContainerRuntime: constants.DefaultRuntimeType,
						PodSandboxImage:  constants.DefaultPodSandboxImage,
					},
					RegisterNode: false,
				},
				TailoredKubeletConfig: &defaultTailedKubeletConfig,
				RegisterNodeNamespace: "test",
			},
		},
	}

	for _, tt := range tests {
		tt.want = tt.edged
		t.Run(tt.name, func(t *testing.T) {
			Register(tt.edged)
			if !reflect.DeepEqual(tt.want.Enable, config.Config.Enable) {
				t.Errorf("TestRegister() = %v, want %v", config.Config.Edged, *tt.want)
			}
		})
	}
}

func TestName(t *testing.T) {
	t.Run("Edged.Name()", func(t *testing.T) {
		if got := (&edged{}).Name(); got != modules.EdgedModuleName {
			t.Errorf("Edged.Name() returned unexpected result. got = %s, want = edged", got)
		}
	})
}

func TestGroup(t *testing.T) {
	t.Run("Edged.Group()", func(t *testing.T) {
		if got := (&edged{}).Group(); got != modules.EdgedGroup {
			t.Errorf("Edged.Group() returned unexpected result. got = %s, want = edged", got)
		}
	})
}

func TestSyncPod_HandlePodListFromMetaManager(t *testing.T) {
	e, _ := newEdged(true, "testnode", "test")
	core.Register(e)

	podConfig := kubeletConfig.NewPodConfig(kubeletConfig.PodConfigNotificationIncremental, nil)
	go e.syncPod(podConfig)

	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testpod1",
		},
		Spec: v1.PodSpec{
			NodeName: "testnode",
		},
	}
	podStr, _ := json.Marshal(pod)
	podStrList := []string{string(podStr)}
	message, _ := beehiveContext.Receive(modules.MetaManagerModuleName)

	respWithResErr := message.NewRespByMessage(&message, podStrList)
	respWithResErr.SetRoute(modules.MetaManagerModuleName, respWithResErr.GetGroup()).SetResourceOperation(model.ResourceTypePod, model.ResponseOperation)
	beehiveContext.Send(modules.EdgedModuleName, *respWithResErr)

	respWithContentMarshalErr := message.NewRespByMessage(&message, make(chan int))
	respWithContentMarshalErr.SetRoute(modules.MetaManagerModuleName, respWithContentMarshalErr.GetGroup())
	beehiveContext.Send(modules.EdgedModuleName, *respWithContentMarshalErr)

	respWithContentUnmarshalErr := message.NewRespByMessage(&message, podStrList)
	respWithContentUnmarshalErr.SetRoute(modules.MetaManagerModuleName, respWithContentUnmarshalErr.GetGroup())
	beehiveContext.Send(modules.EdgedModuleName, *respWithContentUnmarshalErr)

	respSuccess := message.NewRespByMessage(&message, podStr)
	respSuccess.SetRoute(modules.MetaManagerModuleName, respSuccess.GetGroup())
	beehiveContext.Send(modules.EdgedModuleName, *respSuccess)

	time.Sleep(5 * time.Second)

	update := podConfig.Updates()
	t.Run("HandlePodListFromMetaManager Succeed", func(t *testing.T) {
		select {
		case u, _ := <-update:
			if u.Pods[0].Name != pod.Name {
				t.Errorf("pod updates chan got unexpected result. got = %s, want = %s", u.Pods[0].Name, pod.Name)
			}
		default:
			t.Error("no pod msg write into pod update chan")
		}
	})
	beehiveContext.Done()
}

func TestSyncPod_HandlePodListFromEdgeController(t *testing.T) {
	e, _ := newEdged(true, "testnode", "test")
	core.Register(e)

	podConfig := kubeletConfig.NewPodConfig(kubeletConfig.PodConfigNotificationIncremental, nil)
	go e.syncPod(podConfig)

	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testpod2",
		},
		Spec: v1.PodSpec{
			NodeName: "testnode",
		},
	}
	podList := []v1.Pod{pod}
	message, _ := beehiveContext.Receive(modules.MetaManagerModuleName)

	respWithContentUnmarshalErr := message.NewRespByMessage(&message, pod)
	respWithContentUnmarshalErr.SetRoute("edgecontroller", respWithContentUnmarshalErr.GetGroup())
	beehiveContext.Send(modules.EdgedModuleName, *respWithContentUnmarshalErr)

	respSuccess := message.NewRespByMessage(&message, podList)
	respSuccess.SetRoute("edgecontroller", respSuccess.GetGroup())
	beehiveContext.Send(modules.EdgedModuleName, *respSuccess)

	time.Sleep(5 * time.Second)

	update := podConfig.Updates()
	t.Run("HandlePodListFromEdgeController Succeed", func(t *testing.T) {
		select {
		case u, _ := <-update:
			if u.Pods[0].Name != pod.Name {
				t.Errorf("pod updates chan got unexpected result. got = %s, want = %s", u.Pods[0].Name, pod.Name)
			}
		default:
			t.Error("no pod msg write into pod update chan")
		}
	})
	beehiveContext.Done()
}

func TestSyncPod_HandlePod(t *testing.T) {
	e, _ := newEdged(true, "testnode", "test")
	core.Register(e)

	podConfig := kubeletConfig.NewPodConfig(kubeletConfig.PodConfigNotificationIncremental, nil)
	go e.syncPod(podConfig)

	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testpod3",
		},
		Spec: v1.PodSpec{
			NodeName: "testnode",
		},
	}

	beehiveContext.Receive(modules.MetaManagerModuleName)
	message := model.NewMessage("").SetRoute(modules.MetaManagerModuleName, modules.MetaGroup)
	messageWithUnmarshalErr := message.SetResourceOperation(e.namespace+"/"+model.ResourceTypePod+"/"+pod.Name, model.InsertOperation).FillBody([]string{"test"})
	beehiveContext.Send(modules.EdgedModuleName, *messageWithUnmarshalErr)

	messageWithInsertOp := message.SetResourceOperation(e.namespace+"/"+model.ResourceTypePod+"/"+pod.Name, model.InsertOperation).FillBody(pod)
	beehiveContext.Send(modules.EdgedModuleName, *messageWithInsertOp)

	messageWithUpdateOp := message.SetResourceOperation(e.namespace+"/"+model.ResourceTypePod+"/"+pod.Name, model.UpdateOperation).FillBody(pod)
	beehiveContext.Send(modules.EdgedModuleName, *messageWithUpdateOp)

	messageWithDeleteOp := message.SetResourceOperation(e.namespace+"/"+model.ResourceTypePod+"/"+pod.Name, model.DeleteOperation).FillBody(pod)
	beehiveContext.Send(modules.EdgedModuleName, *messageWithDeleteOp)

	time.Sleep(5 * time.Second)

	update := podConfig.Updates()
	t.Run("HandlePod Succeed", func(t *testing.T) {
		select {
		case u, _ := <-update:
			if u.Pods[0].Name != pod.Name {
				t.Errorf("pod updates chan got unexpected result. got = %s, want = %s", u.Pods[0].Name, pod.Name)
			}
			return
		default:
			t.Error("no pod msg write into pod update chan")
		}
	})
	beehiveContext.Done()
}
