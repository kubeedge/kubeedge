package loadbalancer

import (
	"math/rand"
	"sync"

	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/core/registry"
)

// RoundRobinStrategy is strategy
type RoundRobinStrategy struct {
	instances []*registry.MicroServiceInstance
	key       string
}

func newRoundRobinStrategy() Strategy {
	return &RoundRobinStrategy{}
}

//ReceiveData receive data
func (r *RoundRobinStrategy) ReceiveData(inv *invocation.Invocation, instances []*registry.MicroServiceInstance, serviceKey string) {
	r.instances = instances
	r.key = serviceKey
}

//Pick return instance
func (r *RoundRobinStrategy) Pick() (*registry.MicroServiceInstance, error) {
	if len(r.instances) == 0 {
		return nil, ErrNoneAvailableInstance
	}

	i := pick(r.key)
	return r.instances[i%len(r.instances)], nil
}

var rrIdxMap = make(map[string]int)
var mu sync.RWMutex

func pick(key string) int {
	mu.RLock()
	i, ok := rrIdxMap[key]
	if !ok {
		mu.RUnlock()
		mu.Lock()
		i, ok = rrIdxMap[key]
		if !ok {
			i = rand.Int()
			rrIdxMap[key] = i
		}
		rrIdxMap[key]++
		mu.Unlock()
		return i
	}

	mu.RUnlock()
	mu.Lock()
	rrIdxMap[key]++
	mu.Unlock()
	return i
}
