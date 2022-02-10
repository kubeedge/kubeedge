package util

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"

	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/client"
)

const (
	EmptyString = ""
	CrdGroup    = "apiextensions.k8s.io"
	CrdVersion  = "v1beta1"
)

var CRDMapper *meta.DefaultRESTMapper

// SyncCrdResource() is used to trigger the synchronization of CustomResourceDefinition resource
func SyncCrdResource() error {
	if _, err := client.GetCRDClient().ApiextensionsV1beta1().CustomResourceDefinitions().Watch(context.TODO(), metav1.ListOptions{}); err != nil {
		return err
	}
	return nil
}

// UpdateCRDMapper() lists all CustomResourceDefinition resources and add special kind-resource correspondences for them
func UpdateCRDMapper() error {
	list, err := client.GetCRDClient().ApiextensionsV1beta1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, crd := range list.Items {
		CRDMapper.AddSpecific(
			schema.GroupVersionKind{Group: CrdGroup, Version: CrdVersion, Kind: crd.Spec.Names.Kind},
			schema.GroupVersionResource{Group: CrdGroup, Version: CrdVersion, Resource: crd.Spec.Names.Plural},
			schema.GroupVersionResource{Group: CrdGroup, Version: CrdVersion, Resource: crd.Spec.Names.Plural},
			meta.RESTScopeNamespace,
		)
		klog.V(4).Infof("for resource: %s, Kind: %s, Plural: %s", crd.Name, crd.Spec.Names.Kind, crd.Spec.Names.Plural)
	}
	klog.V(4).Infof("The Kind-Resource relationship of all CRD resources has been updated")
	return nil
}

// MetaType is generally consisted of apiversion, kind like:
// {
// apiVersion: apps/v1
// kind: Deployments
// }
// TODO: support crd
func SetMetaType(obj runtime.Object) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	//gvr,_,_ := apiserverlite.ParseKey(accessor.GetSelfLink())
	kinds, _, err := scheme.Scheme.ObjectKinds(obj)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	gvk := kinds[0]
	obj.GetObjectKind().SetGroupVersionKind(gvk)
	klog.V(4).Infof("[metaserver]successfully set MetaType for obj %v, %+v", obj.GetObjectKind(), accessor.GetName())
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
		"nodes":                        "Node",
		"services":                     "Service",
		"podstatus":                    "PodStatus",
		"nodestatus":                   "NodeStatus",
		"customresourcedefinitions":    "CustomResourceDefinition",
		"customresourcedefinitionlist": "CustomResourceDefinitionList",
	}
	if v, isUnusual := unusualResourceToKind[r]; isUnusual {
		return v
	}
	if CRDMapper != nil {
		gvk, err := CRDMapper.KindFor(schema.GroupVersionResource{Resource: r})
		if err == nil && !gvk.Empty() {
			return gvk.Kind
		}
	}
	k := strings.Title(r)
	switch {
	case strings.HasSuffix(k, "ies"):
		return strings.TrimSuffix(k, "ies") + "y"
	case strings.HasSuffix(k, "es"):
		return strings.TrimSuffix(k, "es")
	case strings.HasSuffix(k, "s"):
		return strings.TrimSuffix(k, "s")
	}
	return k
}

func UnsafeKindToResource(k string) string {
	if len(k) == 0 || len(k) == 1 {
		return k
	}
	unusualKindToResource := map[string]string{
		"Endpoints":                    "endpoints",
		"PodStatus":                    "podstatus",
		"NodeStatus":                   "nodestatus",
		"CustomResourceDefinition":     "customresourcedefinitions",
		"CustomResourceDefinitionList": "customresourcedefinitionlist",
	}
	if v, isUnusual := unusualKindToResource[k]; isUnusual {
		return v
	}
	if CRDMapper != nil {
		mapping, err := CRDMapper.RESTMapping(schema.GroupKind{Group: CrdGroup, Kind: k}, CrdVersion)
		if err == nil && mapping != nil && !mapping.Resource.Empty() {
			return mapping.Resource.Resource
		}
	}
	r := strings.ToLower(k)
	switch string(r[len(r)-1]) {
	case "s":
		return r + "es"
	case "y":
		// Rule: When a word ends with y, and the penultimate letter is a vowel, the plural form should change "y" to "ies"
		if string(r[len(r)-2]) == "a" || string(r[len(r)-2]) == "e" || string(r[len(r)-2]) == "i" ||
			string(r[len(r)-2]) == "o" || string(r[len(r)-2]) == "u" {
			return r + "s"
		}
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

// GetMessageUID returns the UID of the object in message
func GetMessageAPIVersion(msg *beehiveModel.Message) string {
	obj, ok := msg.Content.(runtime.Object)
	if ok {
		return obj.GetObjectKind().GroupVersionKind().GroupVersion().String()
	}
	return ""
}

// GetMessageUID returns the UID of the object in message
func GetMessageResourceType(msg *beehiveModel.Message) string {
	obj, ok := msg.Content.(runtime.Object)
	if ok {
		return obj.GetObjectKind().GroupVersionKind().Kind
	}
	return ""
}
