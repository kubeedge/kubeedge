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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/types"
)

var defaultConf = v1alpha1.NewDefaultCloudCoreConfig()
var UC *UpstreamController

func TestMain(m *testing.M) {
	defaultConf.Modules.EdgeController.Enable = true
	var err error
	kubeClient := fake.NewSimpleClientset()
	UC, err = NewUpstreamController(defaultConf.Modules.EdgeController, informers.NewSharedInformerFactory(kubeClient, 0))
	if err != nil {
		panic(err)
	}
	UC.kubeClient = kubeClient
	UC.eventChan = make(chan model.Message, 20)
	go UC.processEvent()
	m.Run()
}

var Events = []*corev1.Event{
	{
		TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "InsertEvent", Namespace: ""},
		Reason:     "insert",
		Message:    "Insert from BIT-CCS group to Kubeedge team",
	},
	{
		TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "UpdateEvent", Namespace: ""},
		Reason:     "update",
		Message:    "Update from BIT-CCS group to Kubeedge team",
	},
	{
		TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "PatchEvent", Namespace: ""},
		Reason:     "insert",
		Message:    "Preparation: Insert from BIT-CCS group to Kubeedge team",
	},
	{
		TypeMeta:   metav1.TypeMeta{Kind: "Event", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "PatchEvent", Namespace: ""},
		Reason:     "patch",
		Message:    "Patch from BIT-CCS group to Kubeedge team",
	},
}

func TestEventReport(t *testing.T) {
	var evtInfo any
	for _, evt := range Events {
		if evt.Reason == "patch" {
			evtInfo = types.EventPatchInfo{
				Event: evt,
				Data:  evt.Message,
			}
		} else {
			evtInfo = evt
		}

		msgContent, err := json.Marshal(evtInfo)
		if err != nil {
			t.Errorf("%s json marshal failed, err: %v", evt.Reason, err)
		}
		msg := model.Message{
			Header:  model.MessageHeader{ID: "bitccs-kubeedge"},
			Router:  model.MessageRoute{Operation: model.InsertOperation},
			Content: string(msgContent),
		}
		UC.eventChan <- msg
		time.Sleep(200 * time.Millisecond)
		fmt.Printf("try to get event %s\n", evt.Name)
		result, err := UC.kubeClient.CoreV1().Events("").Get(context.Background(), evt.Name, metav1.GetOptions{})
		fmt.Printf("get event %s finished\n", evt.Name)
		if result == nil {
			t.Errorf("query %s result got nil", evt.Name)
		} else if result.Name != evt.Name {
			t.Errorf("Event name mismatch, expected %s, got %s", evt.Name, result.Name)
		}
	}
}
