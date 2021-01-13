package apiserverlite

import (
	"context"
	"fmt"
	v2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/pkg/apiserverlite/util"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
	"strings"
)

func KeyFunc(obj runtime.Object) string{
	key ,err := KeyFuncObj(obj)
	if err !=nil{
		klog.Errorf("failed to parse key from an obj:%v",err)
		return ""
	}
	return key
}

// KeyFuncObj generated key from obj
func KeyFuncObj(obj runtime.Object) (string, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "",err
	}
	var key string
	key += "/"
	gvk := obj.GetObjectKind()
	if gvk.GroupVersionKind().Empty(){
		return "", fmt.Errorf("could not get group/version/kind information in obj")
	}
	group := gvk.GroupVersionKind().Group
	version := gvk.GroupVersionKind().Version
	resources := util.UnsafeKindToResource(gvk.GroupVersionKind().Kind)
	namespaces := accessor.GetNamespace()
	name := accessor.GetName()

	if group == "" {
		group = v2.GroupCore
	}

	key += group + "/" + version + "/" + resources + "/"
	if namespaces != ""{
		key += namespaces + "/"
	}else{
		key += v2.NullNamespace + "/"
	}
	//if name == ""{
	//	return "",fmt.Errorf("could not get name information in obj, selflink(%v)",accessor.GetSelfLink())
	//}
	key += name
	key = strings.TrimSuffix(key,"/")
	return key,nil
}

// KeyFuncReq generate key from req context
func KeyFuncReq(ctx context.Context,_ string)(string,error){
	info , ok := apirequest.RequestInfoFrom(ctx)
	var key string
	if ok && info.IsResourceRequest{
		key = "/"
		switch info.APIPrefix{
		case "api":
			key += v2.GroupCore + "/"
		case "apis":
			if info.APIGroup==""{
				return "",fmt.Errorf("failed to get key from request info")
			}
			key += info.APIGroup + "/"
		default:
			return "",fmt.Errorf("failed to get key from request info")
		}
		key += info.APIVersion + "/"
		key += info.Resource +"/"
		if info.Namespace != "" {
			key += info.Namespace + "/"
		}
		if info.Name != ""{
			if info.Namespace == ""{
				key += v2.NullNamespace + "/"
			}
			key += info.Name
		}
	}else{
		return "",fmt.Errorf("no request info in context")
	}
	key = strings.TrimSuffix(key,"/")
	//klog.Infof("[apiserver-lite]get a req, key:%v",key)
	return key, nil

}

func KeyRootFunc(ctx context.Context)string{
	key ,err := KeyFuncReq(ctx,"")
	if err !=nil {
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
func ParseKey(key string)(gvr schema.GroupVersionResource,namespace string,name string) {
	sl := key
	for strings.HasSuffix(sl, "/") {
		sl = strings.TrimSuffix(sl, "/")
	}
	if sl == "" {
		return
	}
	slices := strings.Split(sl,"/")
	length := len(slices)
	if len(slices) == 0 || slices[0] != ""{
		//klog.Errorf("[apiserver-lite]falied to parse key: format error, %v",key)
		return
	}
	var (
		groupIndex  = 1
		versionIndex = 2
		resourceIndex = 3
		namespaceIndex = 4
		nameIndex = 5
	)
	IndexCheck(length,&groupIndex,&versionIndex,&resourceIndex,&namespaceIndex,&nameIndex)

	group := slices[groupIndex]
	if group == v2.GroupCore {
		group = ""
	}

	gv , err := schema.ParseGroupVersion(group + "/" + slices[versionIndex])
	if err!=nil{
		klog.Error(err)
		return
	}

	gvr = gv.WithResource(slices[resourceIndex])
	namespace = slices[namespaceIndex]
	if namespace == v2.NullNamespace{
		namespace = ""
	}
	name = slices[nameIndex]
	if name == v2.NullName{
		name = ""
	}

	return gvr,namespace,name
}

// force set index to 0 if out of range
func IndexCheck(length int, indexes ...*int) {
	for _,index := range indexes {
		if *index >= length {
			*index = 0
		}
	}
	return
}
