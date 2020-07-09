package dao

import (
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

const (
	CacheTableName = "cache"
)

// Cache metadata object
type Cache struct {
	ID int64 `orm:"pk; auto; column(id)"`
	// TODO 确认resource namespace name字段长度
	UA        string `orm:"column(ua); size(256);"`
	Resource  string `orm:"column(resource); size(256)"`
	Namespace string `orm:"column(namespace); size(256)"`
	Name      string `orm:"column(name); size(256)"`
	Value     string `orm:"column(value); null; type(text)"`
}

func (cache *Cache) TableUnique() [][]string {
	return [][]string{
		[]string{"ua", "resource", "namespace", "name"},
	}
}

func DeleteCache(ua, resource, namespace, name string) error {
	qs := dbm.DBAccess.QueryTable(CacheTableName).Filter("ua", ua)
	if resource != "" {
		qs = qs.Filter("resource", resource)
	}
	if namespace != "" {
		qs = qs.Filter("namespace", namespace)
	}
	if name != "" {
		qs = qs.Filter("name", name)
	}
	num, err := qs.Delete()
	klog.V(4).Infof("Delete Cache affected Num: %d, %v", num, err)
	return err
}

func InsertOrUpdate(cache *Cache) error {
	_, err := dbm.DBAccess.Raw("INSERT OR REPLACE INTO cache (ua, resource, namespace, name, value) VALUES (?,?,?,?,?)", cache.UA, cache.Resource, cache.Namespace, cache.Name, cache.Value).Exec() // will update all field
	klog.V(4).Infof("Update result %v", err)
	return err
}

func QueryCacheList(ua, resource, namespace string) ([]string, error) {
	caches := new([]Cache)
	qs := dbm.DBAccess.QueryTable(CacheTableName).Filter("ua", ua).Filter("resource", resource)
	if namespace != "" {
		qs = qs.Filter("namespace", namespace)
	}
	_, err := qs.All(caches)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, v := range *caches {
		result = append(result, v.Value)
	}
	return result, nil
}

func Query(ua, resource, namespace, name string) (string, error) {
	caches := new([]Cache)
	qs := dbm.DBAccess.QueryTable(CacheTableName).Filter("ua", ua).Filter("resource", resource).Filter("name", name)
	if namespace != "" {
		qs = qs.Filter("namespace", namespace)
	}
	_, err := qs.All(caches)
	if err != nil {
		return "", err
	}
	return (*caches)[0].Value, nil
}
