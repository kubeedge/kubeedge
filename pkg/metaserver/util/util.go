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
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"

	kubeedgescheme "github.com/kubeedge/api/client/clientset/versioned/scheme"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
)

// MetaType is generally consisted of apiversion, kind like:
// {
// apiVersion: apps/v1
// kind: Deployments
// }
func SetMetaType(obj runtime.Object) error {
	if obj == nil {
		return fmt.Errorf("object is nil")
	}

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	currentGVK := obj.GetObjectKind().GroupVersionKind()
	if currentGVK.Kind != "" && currentGVK.Version != "" {
		return nil
	}

	gvk, err := objectGVKFromSchemes(obj)
	if err != nil {
		return err
	}

	obj.GetObjectKind().SetGroupVersionKind(gvk)
	klog.V(6).Infof("[metaserver]successfully set MetaType for obj %v, %+v", obj.GetObjectKind(), accessor.GetName())
	return nil
}

// SetMetaTypeByResource makes a best effort to inject Kind/APIVersion.
// It first resolves through registered schemes, then falls back to resource-derived kind.
func SetMetaTypeByResource(obj runtime.Object, resource string) error {
	err := SetMetaType(obj)
	if err == nil {
		return nil
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Kind != "" {
		return nil
	}

	kind := UnsafeResourceToKind(strings.ToLower(resource))
	if kind == "" {
		return err
	}

	gvk.Kind = kind
	obj.GetObjectKind().SetGroupVersionKind(gvk)
	klog.V(4).Infof("[metaserver]set Kind by resource fallback for %T, resource=%s, kind=%s", obj, resource, kind)
	return nil
}

func objectGVKFromSchemes(obj runtime.Object) (schema.GroupVersionKind, error) {
	gvk, err := objectGVKFromScheme(k8sscheme.Scheme, obj)
	if err == nil {
		return gvk, nil
	}

	kubeEdgeGVK, kubeEdgeErr := objectGVKFromScheme(kubeedgescheme.Scheme, obj)
	if kubeEdgeErr == nil {
		return kubeEdgeGVK, nil
	}

	return schema.GroupVersionKind{}, fmt.Errorf("failed to resolve GroupVersionKind from k8s scheme (%v) and kubeedge scheme (%v)", err, kubeEdgeErr)
}

func objectGVKFromScheme(s *runtime.Scheme, obj runtime.Object) (schema.GroupVersionKind, error) {
	kinds, _, err := s.ObjectKinds(obj)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	if len(kinds) == 0 {
		return schema.GroupVersionKind{}, fmt.Errorf("no kind is registered for object type %T", obj)
	}

	currentGVK := obj.GetObjectKind().GroupVersionKind()
	for _, gvk := range kinds {
		if currentGVK.Kind != "" && gvk.Kind != currentGVK.Kind {
			continue
		}
		if currentGVK.Group != "" && gvk.Group != currentGVK.Group {
			continue
		}
		if currentGVK.Version != "" && gvk.Version != currentGVK.Version {
			continue
		}
		return gvk, nil
	}

	return kinds[0], nil
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
		"node":                         "Node",
		"namespaces":                   "Namespace",
		"namespace":                    "Namespace",
		"services":                     "Service",
		"service":                      "Service",
		"configmaps":                   "ConfigMap",
		"configmap":                    "ConfigMap",
		"secrets":                      "Secret",
		"secret":                       "Secret",
		"pods":                         "Pod",
		"pod":                          "Pod",
		"leases":                       "Lease",
		"lease":                        "Lease",
		"events":                       "Event",
		"event":                        "Event",
		"podstatus":                    "PodStatus",
		"nodestatus":                   "NodeStatus",
		"podpatch":                     "PodPatch",
		"nodepatch":                    "NodePatch",
		"devicemodels":                 "DeviceModel",
		"devicemodel":                  "DeviceModel",
		"devices":                      "Device",
		"device":                       "Device",
		"devicestatuses":               "DeviceStatus",
		"devicestatus":                 "DeviceStatus",
		"objectsyncs":                  "ObjectSync",
		"objectsync":                   "ObjectSync",
		"clusterobjectsyncs":           "ClusterObjectSync",
		"clusterobjectsync":            "ClusterObjectSync",
		"customresourcedefinitions":    "CustomResourceDefinition",
		"customresourcedefinitionlist": "CustomResourceDefinitionList",
		"serviceaccounttoken":          "ServiceAccountToken",
		"certificatesigningrequest":    "CertificateSigningRequest",
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

// ParseResourcePath parses resource path to resource type and resource id.
// Supported examples:
// - <namespace>/<resourceType>[/resourceID]
// - node/<node>/<namespace>/<resourceType>[/resourceID]
func ParseResourcePath(resource string) (string, string) {
	trimmed := strings.Trim(resource, constants.ResourceSep)
	if trimmed == "" {
		return "", ""
	}

	tokens := strings.Split(trimmed, constants.ResourceSep)
	if tokens[0] == beehiveModel.ResourceTypeNode {
		// node/<node>/<resourceType> OR node/<node>/<namespace>/<resourceType>[/resourceID]
		switch len(tokens) {
		case 3:
			return tokens[2], ""
		case 4:
			return tokens[3], ""
		case 5:
			return tokens[3], tokens[4]
		default:
			return "", ""
		}
	}

	// <namespace>/<resourceType>[/resourceID]
	switch len(tokens) {
	case 2:
		return tokens[1], ""
	case 3:
		return tokens[1], tokens[2]
	default:
		return "", ""
	}
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
