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
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

var (
	TestOldPodObject = &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("commontestpod"),
			Name:      "TestPod",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "TestPodContainer",
					Image: "busybox",
				},
			},
		},
	}

	TestNewPodObject = &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("commontestpod"),
			Name:      "TestPod",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "TestPodContainer",
					Image: "nginx",
				},
			},
		},
	}

	TestDeletingPodObject = &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("commontestpod"),
			Name:      "TestPod",
			Namespace: "default",
			DeletionTimestamp: &metav1.Time{
				Time: time.Now().Add(1 * time.Minute),
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "TestPodContainer",
					Image: "nginx",
				},
			},
		},
	}
)

func TestCommonResourceEventHandler_obj2Event(t *testing.T) {
	type fields struct {
		events chan watch.Event
	}
	type args struct {
		t   watch.EventType
		obj interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			"TestCommonResourceEventHandler_obj2Event(): Case 1: Test with Pod",
			fields{
				events: make(chan watch.Event, 1),
			},
			args{
				watch.Added,
				TestOldPodObject,
			},
		},
		{
			"TestCommonResourceEventHandler_obj2Event(): Case 2: Test with string",
			fields{
				events: make(chan watch.Event, 1),
			},
			args{
				watch.Added,
				"Hello Kubeedge",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CommonResourceEventHandler{
				events: tt.fields.events,
			}
			c.obj2Event(tt.args.t, tt.args.obj)
			if reflect.TypeOf(tt.args.obj).Kind() == reflect.String {
				return
			}
			obj := <-c.events
			if !reflect.DeepEqual(obj.Type, tt.args.t) || !reflect.DeepEqual(obj.Object, tt.args.obj) {
				t.Errorf("TestCommonResourceEventHandler_obj2Event() failed. got: %v/%v, want %v/%v", obj.Type, obj.Object, tt.args.t, tt.args.obj)
			}
		})
	}
}

func TestCommonResourceEventHandler_OnAdd(t *testing.T) {
	type fields struct {
		events chan watch.Event
	}
	type args struct {
		obj interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			"TestCommonResourceEventHandler_OnAdd(): Case 1: Add Pod",
			fields{
				events: make(chan watch.Event, 1),
			},
			args{
				TestOldPodObject,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CommonResourceEventHandler{
				events: tt.fields.events,
			}
			c.OnAdd(tt.args.obj)
			obj := <-c.events
			if !reflect.DeepEqual(watch.Added, obj.Type) || !reflect.DeepEqual(obj.Object, tt.args.obj) {
				t.Errorf("TestCommonResourceEventHandler_OnAdd() failed. got: %v/%v, want %v/%v", obj.Type, obj.Object, watch.Added, tt.args.obj)
			}
		})
	}
}

func TestCommonResourceEventHandler_OnUpdate(t *testing.T) {
	type fields struct {
		events chan watch.Event
	}
	type args struct {
		oldObj interface{}
		newObj interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			"TestCommonResourceEventHandler_OnUpdate(): Case 1: Test with Pod",
			fields{
				events: make(chan watch.Event, 1),
			},
			args{
				TestOldPodObject,
				TestNewPodObject,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CommonResourceEventHandler{
				events: tt.fields.events,
			}
			c.OnUpdate(tt.args.oldObj, tt.args.newObj)
			obj := <-c.events
			if !reflect.DeepEqual(watch.Modified, obj.Type) || !reflect.DeepEqual(obj.Object, tt.args.newObj) {
				t.Errorf("TestCommonResourceEventHandler_OnUpdate() failed. got: %v/%v, want %v/%v", obj.Type, obj.Object, watch.Modified, tt.args.newObj)
			}
		})
	}
}

func TestCommonResourceEventHandler_OnDelete(t *testing.T) {
	type fields struct {
		events chan watch.Event
	}
	type args struct {
		obj interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			"TestCommonResourceEventHandler_OnDelete(): Case 1: Delete Pod",
			fields{
				events: make(chan watch.Event, 1),
			},
			args{
				TestOldPodObject,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CommonResourceEventHandler{
				events: tt.fields.events,
			}
			c.OnDelete(tt.args.obj)
			obj := <-c.events
			if !reflect.DeepEqual(watch.Deleted, obj.Type) || !reflect.DeepEqual(obj.Object, tt.args.obj) {
				t.Errorf("TestCommonResourceEventHandler_Delete() failed. got: %v/%v, want %v/%v", obj.Type, obj.Object, watch.Deleted, tt.args.obj)
			}
		})
	}
}

func TestNewCommonResourceEventHandler(t *testing.T) {
	type args struct {
		events chan watch.Event
	}
	ch := make(chan watch.Event, 1)
	tests := []struct {
		name string
		args args
		want *CommonResourceEventHandler
	}{
		{
			"TestNewCommonResourceEventHandler(): Case 1: New with events",
			args{
				events: ch,
			},
			&CommonResourceEventHandler{
				events: ch,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCommonResourceEventHandler(tt.args.events); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCommonResourceEventHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}
