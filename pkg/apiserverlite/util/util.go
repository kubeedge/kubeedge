package util

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"strings"
)
const(
	EmptyString = ""
)

// TODO: selflink will be removed at k8s v1.21, choose another way to recover MetaType information
// MetaType is generally consisted of apiversion, kind like:
// {
// apiVersion: apps/v1
// kind: Deployments
// }
// And we add these information here from selflink, and it is like below:
// 1.selflink of an obj
// 0/1  /2 /3         /4      /5   /6
//  /api/v1/namespaces/default/pods/pod-foo
// 2.selflink of a List obj:
// 0/1   /2   /3 /4
//  /apis/apps/v1/deployments
func SetMetaType(obj runtime.Object)error{
	accessor, err:= meta.Accessor(obj)
	if err !=nil{
		return err
	}
	//gvr,_,_ := apiserverlite.ParseKey(accessor.GetSelfLink())
	kinds,_,err := scheme.Scheme.ObjectKinds(obj)
	if err !=nil{
		return fmt.Errorf("%v",err)
	}
	//if gvr.Empty(){
	//	return fmt.Errorf("failed to parse selflink(%v), objName(%v)",accessor.GetSelfLink(),accessor.GetName())
	//	accessor.
	//}
	gvk := kinds[0]
	obj.GetObjectKind().SetGroupVersionKind(gvk)
	klog.V(4).Infof("[apiserver-lite]successfully set MetaType for obj %v, %+v", obj.GetObjectKind(), accessor.GetName())
	return nil
}


// Sometimes, we need guess kind according to resource:
// 1. In most cases, is like pods to Pod,
// 2. In some unusual cases, requires special treatment like endpoints to Endpoints
func UnsafeResourceToKind(r string) string{
	if len(r)==0{
		return r
	}
	unusualResourceToKind := map[string]string{
		"endpoints":"Endpoints",
		"nodes":"Node",
		"services":"Service",
	}
	if v, isUnusual := unusualResourceToKind[r]; isUnusual {
		return v
	}
	k := strings.Title(r)
	switch{
	case strings.HasSuffix(k,"ies"):
		return strings.TrimSuffix(k,"ies") + "y"
	case strings.HasSuffix(k,"es"):
		return strings.TrimSuffix(k,"es")
	case strings.HasSuffix(k,"s"):
		return strings.TrimSuffix(k,"s")
	}
	return k
}

func UnsafeKindToResource(k string)string{
	if len(k) == 0 {
		return k
	}
	unusualKindToResource := map[string]string{
		"Endpoints":"endpoints",
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
	switch obj.GetObjectKind().GroupVersionKind().Kind{
	case "Pod":
		metadata, err := meta.Accessor(obj)
		if err != nil {
			return nil, nil, err
		}
		setMap := make(map[string]string)
		if metadata.GetName() != ""{
			setMap["metadata.name"] = metadata.GetName()
		}
		if metadata.GetNamespace() != ""{
			setMap["metadata.namespaces"] = metadata.GetNamespace()
		}
		unstrObj, ok := obj.(*unstructured.Unstructured)
		if ok {
			value, found, err := unstructured.NestedString(unstrObj.Object,"spec","nodeName")
			if found && err ==nil{
				setMap["spec.nodeName"] = value
			}
		}
		return labels.Set(metadata.GetLabels()),fields.Set(setMap),nil
	default :
		return storage.DefaultNamespaceScopedAttr(obj)
	}
}

