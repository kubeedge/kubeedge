package v2

import (
	"github.com/astaxie/beego/orm"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//constant metatable name reference
const (
	NewMetaTableName = "meta_v2"
	WatchCacheTableName = "watchcache"

	// column name
	KEY = "Key"
	GVR = "GroupVersionResource"
	NS = "Namespace"
	NAME = "Name"
	RV = "ResourceVersion"

	NullNamespace = "null"
	GroupCore = "core"
	NullName = "null"
)
// MetaV2 record k8s api object
type MetaV2 struct {
	// Key is the primary key of a line record, format like k8s obj key in etcd:
	// /Group/Version/Resources/Namespace/Name
	//0/1   /2 /3   /4           /5
	// /core/v1/pods/{namespaces}/{name}									normal obj
	// /core/v1/pods/{namespaces} 											List obj
	// /extensions/v1beta1/ingresses/{namespaces}/{name}				 	normal obj
	// /storage.k8s.io/v1beta1/csidrivers/null/{name} 					 	cluster scope obj
	Key string `orm:"column(key); size(256); pk"`
	// GroupVersionResource are set buy gvr.String() like "/v1, Resource=endpoints"
	GroupVersionResource   string `orm:"column(groupversionresource); size(256);"`
	// Namespace is the namespace of an api object, and set as metadata.namespace
	Namespace  string `orm:"column(namespace); size(256)"`
	// Name is the name of api object, and set as metadata.name
	Name   string `orm:"column(name); size(256)"`
	// ResourceVersion is the resource version of the obj, and set as metadata.resourceVersion
	ResourceVersion uint64`orm:"column(resourceversion); size(256)"`
	// Value is the api object in json format
	// TODO: change to []byte
	Value string `orm:"column(value); null; type(text)"`
}

// List a slice of raw data by Group Version Resource Namespace Name
func RawMetaByGVRNN(gvr schema.GroupVersionResource,namespace string,name string)(*[]MetaV2,error){
	objs := new([]MetaV2)
	var err error
	// TODO: use getCondition
	//cond := getCondition(gvr,namespace,name)
	//klog.Infof("cond:%+v",cond)
	//_,err = dbm.DBAccess.QueryTable(NewMetaTableName).SetCond(cond).All(objs)
	if gvr.Empty(){
		_,err = dbm.DBAccess.QueryTable(NewMetaTableName).All(objs)
	} else{
		switch namespace{
		case NullNamespace,"":
			switch name {
			case NullName,"":
				_, err = dbm.DBAccess.QueryTable(NewMetaTableName).Filter(GVR,gvr.String()).All(objs)
			default:
				_, err = dbm.DBAccess.QueryTable(NewMetaTableName).Filter(GVR,gvr.String()).Filter(NAME,name).All(objs)
			}
		default:
			switch name {
			case NullName,"":
				_, err = dbm.DBAccess.QueryTable(NewMetaTableName).Filter(GVR,gvr.String()).Filter(NS,namespace).All(objs)
			default:
				_, err = dbm.DBAccess.QueryTable(NewMetaTableName).Filter(GVR,gvr.String()).Filter(NS,namespace).Filter(NAME,name).All(objs)
			}
		}

	}
	if err != nil {
		  return nil, err
	}
	return objs, nil
}
func getCondition(gvr schema.GroupVersionResource, namespace string, name string) *orm.Condition{
	cond := orm.NewCondition()
	cond.And(GVR,gvr.String())
	if namespace != NullNamespace {
		cond.And(NS,namespace)
	}
	if name != NullName {
		cond.And(NAME,name)
	}
	return cond
}
