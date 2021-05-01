/*
Copyright 2021 The KubeEdge Authors.

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
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/utils"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

func TestPodManager_isPodUpdated(t *testing.T) {
	type args struct {
		old *CachePod
		new *v1.Pod
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"TestPodManager_isPodUpdated(): Case 1: check differet pod",
			args{
				&CachePod{
					ObjectMeta: TestOldPodObject.ObjectMeta,
					Spec:       TestOldPodObject.Spec,
				},
				TestNewPodObject,
			},
			true,
		},
		{
			"TestPodManager_isPodUpdated(): Case 2: check same pod",
			args{
				&CachePod{
					ObjectMeta: TestOldPodObject.ObjectMeta,
					Spec:       TestOldPodObject.Spec,
				},
				TestOldPodObject,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &PodManager{
				realEvents:   make(chan watch.Event, 1),
				mergedEvents: make(chan watch.Event, 1),
				pods:         sync.Map{},
			}
			if got := pm.isPodUpdated(tt.args.old, tt.args.new); got != tt.want {
				t.Errorf("PodManager.isPodUpdated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodManager_merge(t *testing.T) {
	type fields struct {
		realEvents   chan watch.Event
		mergedEvents chan watch.Event
		pods         *CachePod
	}
	type args struct {
		event watch.Event
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			"TestPodManager_merge(): Case 1: Add pod, deletiontimestamp is nil",
			fields{
				realEvents:   make(chan watch.Event, 1),
				mergedEvents: make(chan watch.Event, 1),
				pods:         nil,
			},
			args{
				event: watch.Event{Type: watch.Added, Object: TestOldPodObject},
			},
		},
		{
			"TestPodManager_merge(): Case 2: Add pod, deletiontimestamp is not nil",
			fields{
				realEvents:   make(chan watch.Event, 1),
				mergedEvents: make(chan watch.Event, 1),
				pods:         nil,
			},
			args{
				event: watch.Event{Type: watch.Added, Object: TestDeletingPodObject},
			},
		},
		{
			"TestPodManager_merge(): Case 3: Modified pod",
			fields{
				realEvents:   make(chan watch.Event, 1),
				mergedEvents: make(chan watch.Event, 1),
				pods:         &CachePod{ObjectMeta: TestOldPodObject.ObjectMeta, Spec: TestOldPodObject.Spec},
			},
			args{
				event: watch.Event{Type: watch.Modified, Object: TestNewPodObject},
			},
		},
		{
			"TestPodManager_merge(): Case 4: Modified pod, new pod is same with old pod",
			fields{
				realEvents:   make(chan watch.Event, 1),
				mergedEvents: make(chan watch.Event, 1),
				pods:         &CachePod{ObjectMeta: TestOldPodObject.ObjectMeta, Spec: TestOldPodObject.Spec},
			},
			args{
				event: watch.Event{Type: watch.Modified, Object: TestOldPodObject},
			},
		},
		{
			"TestPodManager_merge(): Case 5: Modified pod, pod not exist",
			fields{
				realEvents:   make(chan watch.Event, 1),
				mergedEvents: make(chan watch.Event, 1),
				pods:         nil,
			},
			args{
				event: watch.Event{Type: watch.Modified, Object: TestNewPodObject},
			},
		},
		{
			"TestPodManager_merge(): Case 6: Deleted pod",
			fields{
				realEvents:   make(chan watch.Event, 1),
				mergedEvents: make(chan watch.Event, 1),
				pods:         &CachePod{ObjectMeta: TestOldPodObject.ObjectMeta, Spec: TestOldPodObject.Spec},
			},
			args{
				event: watch.Event{Type: watch.Deleted, Object: TestOldPodObject},
			},
		},
		{
			"TestPodManager_merge(): Case 7: Invalid event type",
			fields{
				realEvents:   make(chan watch.Event, 1),
				mergedEvents: make(chan watch.Event, 1),
				pods:         nil,
			},
			args{
				event: watch.Event{Type: "InvalidType", Object: TestOldPodObject},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &PodManager{
				realEvents:   tt.fields.realEvents,
				mergedEvents: tt.fields.mergedEvents,
				pods:         sync.Map{},
			}

			if tt.fields.pods != nil {
				pm.pods.Store(tt.fields.pods.GetUID(), tt.fields.pods)
			}

			go pm.merge()
			tt.fields.realEvents <- tt.args.event
		})
	}
}

func TestPodManager_Events(t *testing.T) {
	type fields struct {
		realEvents   chan watch.Event
		mergedEvents chan watch.Event
	}
	mergeEventsCh := make(chan watch.Event, 1)
	tests := []struct {
		name   string
		fields fields
		want   chan watch.Event
	}{
		{
			"TestPodManager_Events(): Case 1",
			fields{
				make(chan watch.Event, 1),
				mergeEventsCh,
			},
			mergeEventsCh,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &PodManager{
				realEvents:   tt.fields.realEvents,
				mergedEvents: tt.fields.mergedEvents,
				pods:         sync.Map{},
			}
			if got := pm.Events(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PodManager.Events() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPodManager(t *testing.T) {
	type args struct {
		kubeClient *kubernetes.Clientset
		namespace  string
		nodeName   string
	}
	config.Config.KubeAPIConfig = v1alpha1.KubeAPIConfig{
		KubeConfig:  fmt.Sprintf("%s/.kube/config", os.Getenv("HOME")),
		QPS:         100,
		Burst:       200,
		ContentType: "application/vnd.kubernetes.protobuf",
	}
	config.Config.Buffer = &v1alpha1.EdgeControllerBuffer{
		ConfigMapEvent: 1024,
	}

	cli, err := utils.KubeClient()
	if err != nil {
		t.Skip("No k8s cluster config file in $HOME/.kube/config, skip it.")
		return
	}

	tests := []struct {
		name string
		args args
	}{
		{
			"TestNewPodManager(): Case 1: with nodename",
			args{
				cli,
				v1.NamespaceAll,
				"nodename",
			},
		},
		{
			"TestNewPodManager(): Case 2: without nodename",
			args{
				cli,
				v1.NamespaceAll,
				"",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NewPodManager(tt.args.kubeClient, tt.args.namespace, tt.args.nodeName)
		})
	}
}
