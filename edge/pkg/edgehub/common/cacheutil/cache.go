package cacheutil

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/model"
	"k8s.io/klog/v2"
)

type EdgeCache struct {
	cacheStore map[string]*model.Message
	cacheIndex []string
	enabled    bool
}

func NewMetaCache() *EdgeCache {
	return &EdgeCache{
		cacheStore: map[string]*model.Message{},
		cacheIndex: []string{},
		enabled:    false,
	}
}

func (ec *EdgeCache) SaveToCache(m *model.Message) error {
	hash := fmt.Sprintf("%s-%s", m.Router.Resource, m.Router.Operation)
	ec.cacheStore[hash] = m
	ec.sortIndex(hash)
	return nil
}
func (ec *EdgeCache) GetCache() map[string]*model.Message {
	return ec.cacheStore
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

func toMd5(v string) string {
	d := []byte(v)
	m := md5.New()
	m.Write(d)
	return hex.EncodeToString(m.Sum(nil))
}
func (ec *EdgeCache) sortIndex(hash string) {
	klog.Errorf("ec cacheIndex: %v", ec.cacheIndex)
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
