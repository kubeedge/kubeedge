package registry

import (
	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/third_party/forked/k8s.io/apimachinery/pkg/util/sets"
	"github.com/hashicorp/go-version"
	"github.com/patrickmn/go-cache"
	"sync"
)

// IndexCache return instances by criteria
type IndexCache struct {
	latestV    map[string]string //save every service's latest version number
	muxLatestV sync.RWMutex

	simpleCache *cache.Cache //save service name and correspond instances

	//key must contain service name, cache key includes label key values
	indexedCache *cache.Cache

	CriteriaStore []map[string]string //all criteria need to be saved in here so that we can update indexedCache, during Set process
	muxCriteria   sync.RWMutex
}

//NewIndexCache create a cache which saves and manage instances
func NewIndexCache() *IndexCache {
	return &IndexCache{
		simpleCache:  cache.New(DefaultExpireTime, 0),
		latestV:      map[string]string{},
		indexedCache: cache.New(DefaultExpireTime, 0),
		muxLatestV:   sync.RWMutex{},
	}
}

//FullCache return all instances
func (ic *IndexCache) FullCache() *cache.Cache { return ic.simpleCache }

//Delete remove one service's instances
func (ic *IndexCache) Delete(k string) {
	ic.simpleCache.Delete(k)
	ic.indexedCache.Delete(k)
}

//Set overwrite instances cache
func (ic *IndexCache) Set(k string, instances []*MicroServiceInstance) {
	latestV, _ := version.NewVersion("0.0.0")
	for _, instance := range instances {
		//update latest version number
		v, _ := version.NewVersion(instance.version())
		if v != nil && latestV.LessThan(v) {
			ic.muxLatestV.Lock()
			ic.latestV[k] = instance.version()
			ic.muxLatestV.Unlock()
			latestV = v
		}

	}
	////TODO update indexed cache
	//ic.muxCriteria.RLock()
	//for _, criteria := range ic.CriteriaStore {
	//	indexKey := ic.getIndexedCacheKey(k, criteria)
	//	result := make([]*MicroServiceInstance, 0)
	//	for _, instance := range instances {
	//		if instance.Has(criteria) {
	//			result = append(result, instance)
	//		}
	//	}
	//	//forcely overwrite indexed cache, that is safe
	//	ic.indexedCache.Set(indexKey, result, 0)
	//}
	//ic.muxCriteria.RUnlock()

	ic.simpleCache.Set(k, instances, 0)

}

//Get return instances cache by criteria
func (ic *IndexCache) Get(k string, tags map[string]string) ([]*MicroServiceInstance, bool) {
	value, ok := ic.simpleCache.Get(k)
	if !ok {
		return nil, false
	}
	if len(tags) == 0 {
		return value.([]*MicroServiceInstance), ok
	}
	//if version is latest, then set it to real version
	ic.setTagsBeforeQuery(k, tags)
	//find from indexed cache first
	indexKey := getIndexedCacheKey(k, tags)
	savedResult, ok := ic.indexedCache.Get(indexKey)
	if !ok {
		//no result, then find it and save result
		instances, _ := value.([]*MicroServiceInstance)
		queryResult := make([]*MicroServiceInstance, 0, len(instances))
		for _, instance := range instances {
			if instance.Has(tags) {
				queryResult = append(queryResult, instance)
			}
		}
		if len(queryResult) == 0 {
			return nil, false
		}

		//ic.indexedCache.Set(indexKey, queryResult, 0)
		//ic.muxCriteria.Lock()
		//ic.CriteriaStore = append(ic.CriteriaStore, tags)
		//ic.muxCriteria.Unlock()
		return queryResult, true
	}
	return savedResult.([]*MicroServiceInstance), ok

}
func (ic *IndexCache) setTagsBeforeQuery(k string, tags map[string]string) {
	ic.muxLatestV.RLock()
	//must set version before query
	if v, ok := tags[common.BuildinTagVersion]; ok && v == common.LatestVersion && ic.latestV[k] != "" {
		tags[common.BuildinTagVersion] = ic.latestV[k]
	}
	ic.muxLatestV.RUnlock()
}

//must combine keys in order, use sets to return sorted list
func getIndexedCacheKey(service string, tags map[string]string) (ss string) {
	ss = "service:" + service
	keys := sets.NewString()
	for k := range tags {
		keys.Insert(k)
	}
	for _, k := range keys.List() {
		ss += "|" + k + ":" + tags[k]
	}
	return
}
