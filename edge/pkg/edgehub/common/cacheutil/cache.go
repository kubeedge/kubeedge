package cacheutil

import (
	"fmt"
	"sync"

	"github.com/kubeedge/beehive/pkg/core/model"
	"k8s.io/klog/v2"
)

type EdgeCache struct {
	cacheStore map[string]*model.Message
	cacheIndex []string
	enabled    bool
	mu         sync.Mutex
}

func NewMetaCache() *EdgeCache {
	return &EdgeCache{
		cacheStore: map[string]*model.Message{},
		cacheIndex: []string{},
		enabled:    false,
	}
}

func (ec *EdgeCache) SaveToCache(m *model.Message) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	hash := fmt.Sprintf("%s-%s-%s", m.GetResource(), m.GetOperation(), m.GetSource())
	ec.cacheStore[hash] = m
	ec.sortIndex(hash)
	return nil
}
func (ec *EdgeCache) GetCache() map[string]*model.Message {
	return ec.cacheStore
}
func (ec *EdgeCache) GetLock() {
	ec.mu.Lock()
}
func (ec *EdgeCache) ReleaseLock() {
	ec.mu.Unlock()
}
func (ec *EdgeCache) RemoveCache(key string) {
	delete(ec.cacheStore, key)
}

func (ec *EdgeCache) GetCacheIndex() []string {
	return ec.cacheIndex
}
func (ec *EdgeCache) SetEnabled(enable bool) {
	ec.enabled = enable
}
func (ec *EdgeCache) IsEnabled() bool {
	return ec.enabled
}
func (ec *EdgeCache) ShiftIndex() {
	ec.cacheIndex = ec.cacheIndex[1:]
}
func (ec *EdgeCache) GetIndexLength() int {
	return len(ec.cacheIndex)
}
func (ec *EdgeCache) GetFirstIndex() string {
	return ec.cacheIndex[0]
}

func (ec *EdgeCache) sortIndex(hash string) {
	klog.Infof("ec cacheIndex: %v", ec.cacheIndex)
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
