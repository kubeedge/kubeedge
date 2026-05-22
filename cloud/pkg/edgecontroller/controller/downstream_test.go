/*
Copyright 2026 The KubeEdge Authors.

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
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/manager"
)

type downstreamMessageLayer struct {
	sendMessages chan model.Message
}

func (m *downstreamMessageLayer) Send(message model.Message) error {
	m.sendMessages <- message
	return nil
}

func (*downstreamMessageLayer) Receive() (model.Message, error) {
	return model.Message{}, nil
}

func (*downstreamMessageLayer) Response(model.Message) error {
	return nil
}

func TestSyncPodSendsDeleteAfterNodeCacheRemoval(t *testing.T) {
	config := &v1alpha1.EdgeController{
		Buffer: &v1alpha1.EdgeControllerBuffer{PodEvent: 4},
	}
	factory := informers.NewSharedInformerFactory(fake.NewSimpleClientset(), 0)
	podManager, err := manager.NewPodManager(config, factory.Core().V1().Pods().Informer())
	if err != nil {
		t.Fatalf("failed to create pod manager: %v", err)
	}

	lc := &manager.LocationCache{}
	lc.UpdateEdgeNode("edge-node")
	lc.DeleteNode("edge-node")
	if lc.IsEdgeNode("edge-node") {
		t.Fatalf("edge node should be removed from location cache")
	}

	messageLayer := &downstreamMessageLayer{sendMessages: make(chan model.Message, 1)}
	dc := &DownstreamController{
		messageLayer: messageLayer,
		podManager:   podManager,
		lc:           lc,
	}
	go dc.syncPod()

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-pod",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		Spec: v1.PodSpec{
			NodeName: "edge-node",
			Containers: []v1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	podManager.Events() <- watch.Event{Type: watch.Modified, Object: pod}
	expectNoPodMessage(t, messageLayer.sendMessages)

	unassignedPod := pod.DeepCopy()
	unassignedPod.Spec.NodeName = ""
	podManager.Events() <- watch.Event{Type: watch.Deleted, Object: unassignedPod}
	expectNoPodMessage(t, messageLayer.sendMessages)

	cloudPod := pod.DeepCopy()
	cloudPod.Spec.NodeName = "cloud-node"
	podManager.Events() <- watch.Event{Type: watch.Deleted, Object: cloudPod}
	expectNoPodMessage(t, messageLayer.sendMessages)

	podManager.Events() <- watch.Event{Type: watch.Deleted, Object: pod}

	select {
	case msg := <-messageLayer.sendMessages:
		if msg.GetOperation() != model.DeleteOperation {
			t.Fatalf("expected operation %q, got %q", model.DeleteOperation, msg.GetOperation())
		}
		wantResource := "node/edge-node/default/" + model.ResourceTypePod + "/test-pod"
		if msg.GetResource() != wantResource {
			t.Fatalf("expected resource %q, got %q", wantResource, msg.GetResource())
		}
	case <-time.After(time.Second):
		t.Fatalf("expected pod delete message")
	}
}

func expectNoPodMessage(t *testing.T, messages <-chan model.Message) {
	t.Helper()
	select {
	case msg := <-messages:
		t.Fatalf("expected no message, got operation %q for %q", msg.GetOperation(), msg.GetResource())
	case <-time.After(100 * time.Millisecond):
	}
}
