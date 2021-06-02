package metaserver

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"

	v2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

func KeyFunc(obj runtime.Object) string {
	key, err := KeyFuncObj(obj)
	if err != nil {
		klog.Errorf("failed to parse key from an obj:%v", err)
		return ""
	}
	return key
}

// KeyFuncObj generated key from obj
func KeyFuncObj(obj runtime.Object) (string, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", err
	}

	objKind := obj.GetObjectKind()
	gvk := objKind.GroupVersionKind()
	if gvk.Empty() {
		return "", fmt.Errorf("could not get group/version/kind information in obj")
	}
	group := gvk.Group
	version := gvk.Version
	resource := util.UnsafeKindToResource(gvk.Kind)
	namespace := accessor.GetNamespace()
	name := accessor.GetName()

	if group == "" {
		group = v2.GroupCore
	}
	if namespace == "" {
		namespace = v2.NullNamespace
	}

	key := fmt.Sprintf("/%s/%s/%s/%s/%s", group, version, resource, namespace, name)
	return key, nil
}

// KeyFuncReq generate key from req context
func KeyFuncReq(ctx context.Context, _ string) (string, error) {
	info, ok := apirequest.RequestInfoFrom(ctx)
	if !ok || !info.IsResourceRequest {
		return "", fmt.Errorf("no request info in context")
	}

	group := ""
	switch info.APIPrefix {
	case "api":
		group = v2.GroupCore
	case "apis":
		if info.APIGroup == "" {
			return "", fmt.Errorf("failed to get key from request info")
		}
		group = info.APIGroup
	default:
		return "", fmt.Errorf("failed to get key from request info")
	}
	version := info.APIVersion
	resource := info.Resource
	namespace := info.Namespace
	name := info.Name
	if namespace == "" {
		namespace = v2.NullNamespace
	}
	if name == "" {
		name = v2.NullName
	}

	key := fmt.Sprintf("/%s/%s/%s/%s/%s", group, version, resource, namespace, name)
	return key, nil
}

func KeyRootFunc(ctx context.Context) string {
	key, err := KeyFuncReq(ctx, "")
	if err != nil {
		panic("fail to get list key!")
	}
	return key
}

// ParseKey parse key to group/version/resource, namespace, name
// Now key format is like below:
// 0/1   /2 /3   /4     /5
//  /core/v1/pods/{namespaces}/{name}
// 0/1  /2/3
//  /app/v1/deployments
// 0/1   /2 /3
//  /core/v1/endpoints
// Remember that ParseKey is not responsible for verifying the validity of the content,
// for example, gvr in key /app/v1111/endpoint will be parsed as {Group:"app", Version:"v1111", Resource:"endpoint"}
func ParseKey(key string) (gvr schema.GroupVersionResource, namespace string, name string) {
	sl := key
	for strings.HasSuffix(sl, "/") {
		sl = strings.TrimSuffix(sl, "/")
	}
	if sl == "" {
		return
	}
	slices := strings.Split(sl, "/")
	length := len(slices)
	if len(slices) == 0 || slices[0] != "" {
		//klog.Errorf("[metaserver]falied to parse key: format error, %v",key)
		return
	}
	var (
		groupIndex     = 1
		versionIndex   = 2
		resourceIndex  = 3
		namespaceIndex = 4
		nameIndex      = 5
	)
	IndexCheck(length, &groupIndex, &versionIndex, &resourceIndex, &namespaceIndex, &nameIndex)

	group := slices[groupIndex]
	if group == v2.GroupCore {
		group = ""
	}

	gv, err := schema.ParseGroupVersion(group + "/" + slices[versionIndex])
	if err != nil {
		klog.Error(err)
		return
	}

	gvr = gv.WithResource(slices[resourceIndex])
	namespace = slices[namespaceIndex]
	if namespace == v2.NullNamespace {
		namespace = ""
	}
	name = slices[nameIndex]
	if name == v2.NullName {
		name = ""
	}

	return gvr, namespace, name
}

// force set index to 0 if out of range
func IndexCheck(length int, indexes ...*int) {
	for _, index := range indexes {
		if *index >= length {
			*index = 0
		}
	}
}
