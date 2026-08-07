package util

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/storage"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"

	kubeedgescheme "github.com/kubeedge/api/client/clientset/versioned/scheme"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
)

// MetaType is generally consisted of apiversion, kind like:
// {
// apiVersion: apps/v1
// kind: Deployments
// }
func SetMetaType(obj runtime.Object) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	gvk, err := getObjectKind(obj)
	if err != nil {
		return err
	}
	obj.GetObjectKind().SetGroupVersionKind(gvk)
	klog.V(6).Infof("[metaserver]successfully set MetaType for obj %v, %+v", obj.GetObjectKind(), accessor.GetName())
	return nil
}

// Sometimes, we need guess kind according to resource:
// 1. In most cases, is like pods to Pod,
// 2. In some unusual cases, requires special treatment like endpoints to Endpoints
func UnsafeResourceToKind(r string) string {
	if len(r) == 0 {
		return r
	}
	unusualResourceToKind := map[string]string{
		"endpoints":                    "Endpoints",
		"endpointslices":               "EndpointSlice",
		"nodes":                        "Node",
		"namespaces":                   "Namespace",
		"services":                     "Service",
		"podstatus":                    "PodStatus",
		"nodestatus":                   "NodeStatus",
		"customresourcedefinitions":    "CustomResourceDefinition",
		"customresourcedefinitionlist": "CustomResourceDefinitionList",
		"leases":                       "Leases",
	}
	if v, isUnusual := unusualResourceToKind[r]; isUnusual {
		return v
	}
	caser := cases.Title(language.Und)
	k := caser.String(r)
	switch {
	case strings.HasSuffix(k, "ies"):
		return strings.TrimSuffix(k, "ies") + "y"
	case strings.HasSuffix(k, "ses"):
		return strings.TrimSuffix(k, "es")
	case strings.HasSuffix(k, "s"):
		return strings.TrimSuffix(k, "s")
	}
	return k
}

func UnsafeKindToResource(k string) string {
	if len(k) == 0 {
		return k
	}
	unusualKindToResource := map[string]string{
		"Endpoints":                    "endpoints",
		"PodStatus":                    "podstatus",
		"NodeStatus":                   "nodestatus",
		"CustomResourceDefinition":     "customresourcedefinitions",
		"CustomResourceDefinitionList": "customresourcedefinitionlist",
		"Leases":                       "leases",
	}
	if v, isUnusual := unusualKindToResource[k]; isUnusual {
		return v
	}
	r := strings.ToLower(k)
	switch string(r[len(r)-1]) {
	case "s":
		return r + "es"
	case "y":
		return strings.TrimSuffix(r, "y") + "ies"
	}

	return r + "s"
}

func UnstructuredAttr(obj runtime.Object) (labels.Set, fields.Set, error) {
	switch obj.GetObjectKind().GroupVersionKind().Kind {
	case "Pod":
		metadata, err := meta.Accessor(obj)
		if err != nil {
			return nil, nil, err
		}
		setMap := make(fields.Set)
		if metadata.GetName() != "" {
			setMap["metadata.name"] = metadata.GetName()
		}
		if metadata.GetNamespace() != "" {
			setMap["metadata.namespaces"] = metadata.GetNamespace()
		}
		unstrObj, ok := obj.(*unstructured.Unstructured)
		if ok {
			value, found, err := unstructured.NestedString(unstrObj.Object, "spec", "nodeName")
			if found && err == nil {
				setMap["spec.nodeName"] = value
			}
		}
		return metadata.GetLabels(), setMap, nil
	default:
		return storage.DefaultNamespaceScopedAttr(obj)
	}
}

// GetMessageAPIVersion returns the API version of the object in message.
func GetMessageAPIVersion(msg *beehiveModel.Message) string {
	obj, ok := msg.Content.(runtime.Object)
	if ok {
		gvk, err := getObjectKind(obj)
		if err != nil {
			return ""
		}
		return gvk.GroupVersion().String()
	}
	return ""
}

// GetMessageResourceType returns the kind of the object in message.
func GetMessageResourceType(msg *beehiveModel.Message) string {
	obj, ok := msg.Content.(runtime.Object)
	if ok {
		gvk, err := getObjectKind(obj)
		if err != nil {
			return ""
		}
		return gvk.Kind
	}
	return ""
}

func getObjectKind(obj runtime.Object) (schema.GroupVersionKind, error) {
	if obj == nil {
		return schema.GroupVersionKind{}, fmt.Errorf("object is nil")
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	if !gvk.Empty() {
		return gvk, nil
	}

	for _, scheme := range []*runtime.Scheme{kubescheme.Scheme, kubeedgescheme.Scheme} {
		kinds, _, err := scheme.ObjectKinds(obj)
		if err == nil && len(kinds) > 0 {
			return kinds[0], nil
		}
	}

	return schema.GroupVersionKind{}, fmt.Errorf("no kind is registered for object %T", obj)
}

type key int

const (
	// applicationKey is the context key for the application.
	applicationIDKey key = iota
)

// WithApplicationID returns a copy of parent in which the applicationID value is set
func WithApplicationID(parent context.Context, appID string) context.Context {
	return context.WithValue(parent, applicationIDKey, appID)
}

// ApplicationIDFrom returns the value of the ApplicationID key on the ctx
func ApplicationIDFrom(ctx context.Context) (string, bool) {
	applicationID, ok := ctx.Value(applicationIDKey).(string)
	return applicationID, ok
}

// ApplicationIDValue returns the value of the applicationID key on the ctx, or the empty string if none
func ApplicationIDValue(ctx context.Context) string {
	applicationID, _ := ApplicationIDFrom(ctx)
	return applicationID
}
