package weightpool

import (
	"sync"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/config"
)

var weightPool *SafePool
var once sync.Once

func init() { once.Do(func() { weightPool = &SafePool{pool: map[string]*Pool{}} }) }

// GetPool returns singleton of weightPool
func GetPool() *SafePool { return weightPool }

// SafePool is a cache for pool of all destination
type SafePool struct {
	sync.RWMutex
	pool map[string]*Pool
}

// Get returns specific pool for key
func (s *SafePool) Get(key string) (*Pool, bool) {
	s.RLock()
	value, ok := s.pool[key]
	s.RUnlock()
	return value, ok
}

// Set can set pool to safe cache
func (s *SafePool) Set(key string, value *Pool) {
	s.Lock()
	s.pool[key] = value
	s.Unlock()
}

// Reset can delete pool for specific key
func (s *SafePool) Reset(key string) {
	s.Lock()
	delete(s.pool, key)
	s.Unlock()
}

/* Weighted Round-Robin Scheduling
http://zh.linuxvirtualserver.org/node/37

while (true) {
  i = (i + 1) mod n;
  if (i == 0) {
     cw = cw - gcd(S);
     if (cw <= 0) {
       cw = max(S);
       if (cw == 0)
         return NULL;
     }
  }
  if (W(Si) >= cw)
    return Si;
}*/

// Pool defines sets of weighted tags
type Pool struct {
	tags []config.RouteTag

	mu  sync.RWMutex
	gcd int
	max int
	i   int
	cw  int
	num int
}

// NewPool returns pool for provided tags
func NewPool(routeTags ...*config.RouteTag) *Pool {
	var total int
	p := &Pool{tags: make([]config.RouteTag, len(routeTags))}
	for i, t := range routeTags {
		if t.Weight > 0 {
			total += t.Weight
			p.refreshGCD(t)
		}
		p.tags[i] = *t
	}

	if total < 100 {
		latestT := config.RouteTag{
			Weight: 100 - total,
			Tags: map[string]string{
				common.BuildinTagVersion: common.LatestVersion,
			},
			Label: common.BuildinLabelVersion,
		}
		p.refreshGCD(&latestT)
		p.tags = append(p.tags, latestT)
	}

	p.num = len(p.tags)
	return p
}

// PickOne returns tag according to its weight
func (p *Pool) PickOne() *config.RouteTag {
	if p.num == 0 || p.max == 0 {
		return nil
	}
	if p.num == 1 {
		return &p.tags[0]
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		p.i = (p.i + 1) % p.num
		if p.i == 0 {
			p.cw = p.cw - p.gcd
			if p.cw <= 0 {
				p.cw = p.max
			}
		}

		if p.tags[p.i].Weight >= p.cw {
			return &p.tags[p.i]
		}
	}
}

func (p *Pool) refreshGCD(t *config.RouteTag) {
	p.gcd = gcd(p.gcd, t.Weight)
	if p.max < t.Weight {
		p.max = t.Weight
	}
}

func gcd(a, b int) int {
	if b == 0 {
		return a
	}
	return gcd(b, a%b)
}
