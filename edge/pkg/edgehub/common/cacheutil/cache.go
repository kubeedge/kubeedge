package cacheutil

import (
	"fmt"
	"sync"

	"github.com/kubeedge/beehive/pkg/core/model"
	"k8s.io/klog/v2"
)

type edgeHub interface {
	SendCacheToCloud(message model.Message) error
}

type EdgeCache struct {
	cacheStore map[string]*model.Message
	cacheIndex []string
	enabled    bool
	edgeHub    edgeHub
	mu         sync.Mutex
}

func NewMetaCache(eh edgeHub) *EdgeCache {
	return &EdgeCache{
		cacheStore: map[string]*model.Message{},
		cacheIndex: []string{},
		enabled:    false,
		edgeHub:    eh,
	}
}

func (ec *EdgeCache) SaveToCache(m *model.Message) error {
	ec.mu.Lock()
	hash := fmt.Sprintf("%s-%s-%s", m.GetResource(), m.GetOperation(), m.GetSource())
	ec.cacheStore[hash] = m
	ec.sortIndex(hash)
	ec.mu.Unlock()
	return nil
}
func (ec *EdgeCache) CacheToCloud() error {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	klog.Infof("start sending cache to cloud")
	for len(ec.cacheIndex) > 0 {
		name := ec.cacheIndex[0]
		err := ec.edgeHub.SendCacheToCloud(*ec.cacheStore[name])
		if err != nil {
			return err
		}
		klog.Infof("successfully send cache %s to cloud", name)
		ec.cacheIndex = ec.cacheIndex[1:]
		delete(ec.cacheStore, name)
	}
	klog.Infof("finished sending cache to cloud")
	return nil
}
func (ec *EdgeCache) GetCache() map[string]*model.Message {
	return ec.cacheStore
}
func (ec *EdgeCache) SetEnabled(enable bool) {
	ec.enabled = enable
}
func (ec *EdgeCache) IsEnabled() bool {
	return ec.enabled
}
func (ec *EdgeCache) sortIndex(hash string) {
	klog.V(4).Infof("ec cacheIndex: %v", ec.cacheIndex)
	for index, item := range ec.cacheIndex {
		if item == hash {
			if (index + 1) != len(ec.cacheIndex) {
				copy(ec.cacheIndex[index:], ec.cacheIndex[index+1:])
				ec.cacheIndex[len(ec.cacheIndex)-1] = hash
			}
			return
		}
	}
	ec.cacheIndex = append(ec.cacheIndex, hash)

}
func (ec *EdgeCache) CleanIndex() {
	ec.cacheIndex = ec.cacheIndex[0:0]
}
