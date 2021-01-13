package cacher

import (
	"fmt"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/apiserver-lite/kubernetes/storage/sqlite"
	"github.com/kubeedge/kubeedge/pkg/apiserverlite"
	"github.com/kubeedge/kubeedge/pkg/apiserverlite/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	cacherstorage "k8s.io/apiserver/pkg/storage/cacher"
	"k8s.io/apiserver/pkg/storage/etcd3"
)
func UnstrIndexFunc(obj interface{}) ([]string, error) {
	unstrObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("not a unstructured obj")
	}
	key,err := apiserverlite.KeyFuncObj(unstrObj)
	if err !=nil{
		return []string{},err
	}
	return []string{key},nil
}
func NewCacher() (storage.Interface,error){
	cacherConfig :=cacherstorage.Config{
		Storage:        sqlite.New(),
		Versioner:      etcd3.Versioner,
		ResourcePrefix: "",
		KeyFunc:        apiserverlite.KeyFuncObj,
		NewFunc: func()runtime.Object{
			unstr := unstructured.Unstructured{}
			return &unstr
		},
		NewListFunc: func()runtime.Object{
			unstrList := unstructured.UnstructuredList{}
			return &unstrList
		},
		GetAttrsFunc: util.UnstructuredAttr,
		// TODO: find appropriate IndexerFuncs or Indexer.
		//IndexerFuncs: map[string]storage.IndexerFunc{"metadata.name": configmap.NameTriggerFunc},
		IndexerFuncs: nil,
		//Indexers: &cache.Indexers{"unstrIndexer": UnstrIndexFunc},
		Indexers: nil,
		Codec: unstructured.UnstructuredJSONScheme,
	}


	cacher, err := cacherstorage.NewCacherFromConfig(cacherConfig)
	if err !=nil{
		return nil,err
	}
	return cacher,nil
}