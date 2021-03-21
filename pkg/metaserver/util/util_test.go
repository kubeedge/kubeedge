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
	type args struct {
		msg *beehiveModel.Message
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestGetMessageAPIVerison(): Case 1: Content pod",
			args: args{
				msg: &beehiveModel.Message{
					Content: &corev1.Pod{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
					},
				}},
			want: "v1",
		},
		{
			name: "TestGetMessageAPIVerison(): Case 2: Content other",
			args: args{
				msg: &beehiveModel.Message{
					Content: "content",
				}},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMessageAPIVerison(tt.args.msg); got != tt.want {
				t.Errorf("GetMessageAPIVerison() = %v, want %v", got, tt.want)
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
		args args
		want string
	}{
		{
			name: "TestGetMessageResourceType(): Case 1: Content pod",
			args: args{
				msg: &beehiveModel.Message{
					Content: &corev1.Pod{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
					},
				}},
			want: UnsafeKindToResource("Pod"),
		},
		{
			name: "TestGetMessageResourceType(): Case 2: Content other",
			args: args{
				msg: &beehiveModel.Message{
					Content: "content",
				}},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMessageResourceType(tt.args.msg); got != tt.want {
				t.Errorf("GetMessageResourceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetMetaType(t *testing.T) {
	type args struct {
		obj runtime.Object
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TestSetMetaType(): Case 1: Set MetaType success",
			args:    args{obj: &appsv1.Deployment{}},
			wantErr: false,
		},
		{
			name:    "TestSetMetaType(): Case 2: Set MetaType fail",
			args:    args{obj: nil},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetMetaType(tt.args.obj); (err != nil) != tt.wantErr {
				t.Errorf("SetMetaType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnsafeKindToResource(t *testing.T) {
	type args struct {
		k string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestUnsafeKindToResource(): Case 1: Endpoints",
			args: args{k: "Endpoints"},
			want: "endpoints",
		},
		{
			name: "TestUnsafeKindToResource(): Case 2: s",
			args: args{k: "Class"},
			want: "classes",
		},
		{
			name: "TestUnsafeKindToResource(): Case 3: y",
			args: args{k: "Copy"},
			want: "copies",
		},
		{
			name: "TestUnsafeKindToResource(): Case 4: other",
			args: args{k: "Service"},
			want: "services",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnsafeKindToResource(tt.args.k); got != tt.want {
				t.Errorf("UnsafeKindToResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnsafeResourceToKind(t *testing.T) {
	type args struct {
		r string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestUnsafeResourceToKind(): Case 1: Endpoint",
			args: args{r: "endpoints"},
			want: "Endpoints",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 2: Node",
			args: args{r: "nodes"},
			want: "Node",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 3: Service",
			args: args{r: "services"},
			want: "Service",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 4: ies",
			args: args{r: "Copies"},
			want: "Copy",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 5: es",
			args: args{r: "Classes"},
			want: "Class",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 6: s",
			args: args{r: "Deployments"},
			want: "Deployment",
		},
		{
			name: "TestUnsafeResourceToKind(): Case 7: other",
			args: args{r: "Other"},
			want: "Other",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnsafeResourceToKind(tt.args.r); got != tt.want {
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
	type args struct {
		obj runtime.Object
	}
	tests := []struct {
		name    string
		args    args
		want    labels.Set
		want1   fields.Set
		wantErr bool
	}{
		{
			name: "TestUnstructuredAttr(): Case 1: Deployment",
			args: args{obj: dep},
			want: dep.GetLabels(),
			want1: map[string]string{
				"metadata.name":      "deployment1",
				"metadata.namespace": "test",
			},
			wantErr: false,
		},
		{
			name: "TestUnstructuredAttr(): Case 2: Pod",
			args: args{obj: uns},
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
			got, got1, err := UnstructuredAttr(tt.args.obj)
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
