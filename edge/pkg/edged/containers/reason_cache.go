package containers

import (
	"sync"

	"github.com/golang/groupcache/lru"
)

// ReasonCache store the failure reason of the latest container start
// in a string, keyed by <podName> or <podName><specContainerName>. The
// goal is to propagate this reason to the container status. This is endeavor is
// "best-effort" for two reasons:
//  1. The cache is not persisted.
//  2. We use an LRU cache to avoid extra garbage collection work. This is means that
//	   some entries may be recycled before a pod has been deleted.
// TODO(random-liu): use more reliable cache which could collect garbage of failed pod
// TODO(random-liu): Move reason cache to somewhere better
type ReasonCache struct {
	lock  sync.Mutex
	cache *lru.Cache
}

// ReasonItem is the cached item in ReasonCache
type ReasonItem struct {
	Err     error
	Message string
}

// maxReasonCache Entries is the cache entry number in lru cache. 1000 is a proper number
// for our 100 pods per node target. If we support more pods per node in the future, we
// may want to increase the number.
const maxReasonCacheEntries = 1000

// NewReasonCache creates an instance of 'ReasonCache'.
func NewReasonCache() *ReasonCache {
	return &ReasonCache{cache: lru.New(maxReasonCacheEntries)}
}

// Add adds error reason into the cache
func (c *ReasonCache) Add(name string, reason error, message string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache.Add(name, ReasonItem{reason, message})
}

// Remove removes error reason from the cache
func (c *ReasonCache) Remove(name string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache.Remove(name)
}

// Get get error reason from the cache. The return values are error reason, error message and
// whether an error reason is found in the cache. If no error reason is found, empty string will
// be returned for error reason and error message.
func (c *ReasonCache) Get(name string) (*ReasonItem, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	value, ok := c.cache.Get(name)
	if !ok {
		return nil, false
	}

	info := value.(ReasonItem)
	return &info, true
}
