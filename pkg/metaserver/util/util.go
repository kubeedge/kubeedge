package util

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
)

const (
	EmptyString = ""
)

var (
	CRDResourceToKind map[string]string
	CRDKindToResource map[string]string
)

func InitCrdMap() error {
	CRDResourceToKind = make(map[string]string)
	CRDKindToResource = make(map[string]string)

	config, err := clientcmd.BuildConfigFromFlags("127.0.0.1:10550", "")
	if err != nil {
		klog.Errorf("Failed to build config, err: %v", err)
		return err
	}
	config.ContentType = runtime.ContentTypeJSON
	apiExtensionClient, err := clientset.NewForConfig(config)
	list, err := apiExtensionClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, crd := range list.Items {
		kind := crd.Spec.Names.Kind
		plural := crd.Spec.Names.Plural
		CRDResourceToKind[plural] = kind
		CRDKindToResource[kind] = plural
	}
	klog.Infof("CRD Resource-Kind map initialization finished.")
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
		"customresourcedefinitions":    "CustomResourceDefinition",
		"customresourcedefinitionlist": "CustomResourceDefinitionList",
	}
	if v, isUnusual := unusualResourceToKind[r]; isUnusual {
		return v
	}
	if v, isCRD := CRDResourceToKind[r]; isCRD {
		return v
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
	if len(k) == 0 {
		return k
	}
	unusualKindToResource := map[string]string{
		"Endpoints":                    "endpoints",
		"CustomResourceDefinition":     "customresourcedefinitions",
		"CustomResourceDefinitionList": "customresourcedefinitionlist",
	}
	if v, isUnusual := unusualKindToResource[k]; isUnusual {
		return v
	}
	if v, isCRD := CRDKindToResource[k]; isCRD {
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

// GetMessageUID returns the UID of the object in message
func GetMessageAPIVerison(msg *beehiveModel.Message) string {
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
		return UnsafeKindToResource(obj.GetObjectKind().GroupVersionKind().Kind)
	}
	return ""
}
