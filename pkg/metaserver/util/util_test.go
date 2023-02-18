package util

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
)

func TestGetMessageAPIVerison(t *testing.T) {
	tests := []struct {
		name string
		msg  *beehiveModel.Message
		want string
	}{
		{
			name: "TestGetMessageAPIVerison(): Case 1: Content pod",
			msg: &beehiveModel.Message{
				Content: &corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
				},
			},
			want: "v1",
		},
		{
			name: "TestGetMessageAPIVerison(): Case 2: Content other",
			msg: &beehiveModel.Message{
				Content: "content",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMessageAPIVersion(tt.msg); got != tt.want {
				t.Errorf("GetMessageAPIVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMessageResourceType(t *testing.T) {
	type args struct {
		msg *beehiveModel.Message
	}
	tests := []struct {
		name string
		msg  *beehiveModel.Message
		want string
	}{
		{
			name: "TestGetMessageResourceType(): Case 1: Content pod",
			msg: &beehiveModel.Message{
				Content: &corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
				},
			},
			want: "Pod",
		},
		{
			name: "TestGetMessageResourceType(): Case 2: Content other",
			msg: &beehiveModel.Message{
				Content: "content",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMessageResourceType(tt.msg); got != tt.want {
				t.Errorf("GetMessageResourceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetMetaType(t *testing.T) {
	tests := []struct {
		name    string
		obj     runtime.Object
		wantErr bool
	}{
		{
			name:    "TestSetMetaType(): Case 1: Set MetaType success",
			obj:     &appsv1.Deployment{},
			wantErr: false,
		},
		{
			name:    "TestSetMetaType(): Case 2: Set MetaType fail",
			obj:     nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetMetaType(tt.obj); (err != nil) != tt.wantErr {
				t.Errorf("SetMetaType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnsafeKindToResource(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{
			name: "TestUnsafeKindToResource(): Case 1: Endpoints",
			args: "Endpoints",
			want: "endpoints",
		},
		{
			name: "TestUnsafeKindToResource(): Case 2: s",
			args: "Class",
			want: "classes",
		},
		{
			name: "TestUnsafeKindToResource(): Case 3: y",
			args: "Copy",
			want: "copies",
		},
		{
			name: "TestUnsafeKindToResource(): Case 4: other",
			args: "Service",
			want: "services",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnsafeKindToResource(tt.args); got != tt.want {
				t.Errorf("UnsafeKindToResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnsafeResourceToKind(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{
			name: "TestUnsafeResourceToKind(): Case 1: Endpoint",
			args: "endpoints",
			want: "Endpoints",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 2: Node",
			args: "nodes",
			want: "Node",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 3: Service",
			args: "services",
			want: "Service",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 4: ies",
			args: "Copies",
			want: "Copy",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 5: es",
			args: "Classes",
			want: "Class",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 6: s",
			args: "Deployments",
			want: "Deployment",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 7: other",
			args: "Other",
			want: "Other",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnsafeResourceToKind(tt.args); got != tt.want {
				t.Errorf("UnsafeResourceToKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnstructuredAttr(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment1",
			Namespace: "test",
			Labels: map[string]string{
				"metadata.name":      "deployment1",
				"metadata.namespace": "test",
			},
		},
	}
	uns := &unstructured.Unstructured{}
	uns.SetName("uns1")
	uns.SetNamespace("test")
	uns.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	})
	uns.SetLabels(map[string]string{
		"metadata.name":       "uns1",
		"metadata.namespaces": "test",
	})
	_ = unstructured.SetNestedField(uns.Object, "node1", "spec", "nodeName")
	tests := []struct {
		name    string
		obj     runtime.Object
		want    labels.Set
		want1   fields.Set
		wantErr bool
	}{
		{
			name: "TestUnstructuredAttr(): Case 1: Deployment",
			obj:  dep,
			want: dep.GetLabels(),
			want1: map[string]string{
				"metadata.name":      "deployment1",
				"metadata.namespace": "test",
			},
			wantErr: false,
		},
		{
			name: "TestUnstructuredAttr(): Case 2: Pod",
			obj:  uns,
			want: uns.GetLabels(),
			want1: map[string]string{
				"metadata.name":       "uns1",
				"metadata.namespaces": "test",
				"spec.nodeName":       "node1",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := UnstructuredAttr(tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnstructuredAttr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnstructuredAttr() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("UnstructuredAttr() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
