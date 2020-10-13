package loadbalancer

import (
	"sync"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/core/registry"
	"github.com/go-chassis/go-chassis/session"
)

var (

	// successiveFailureCount success and failure count
	successiveFailureCount      map[string]int
	successiveFailureCountMutex sync.RWMutex
)

func init() {
	successiveFailureCount = make(map[string]int)
}

//DeleteSuccessiveFailureCount deleting cookie from failure count map
func DeleteSuccessiveFailureCount(cookieValue string) {
	successiveFailureCountMutex.Lock()
	//	successiveFailureCount[ep] = 0
	delete(successiveFailureCount, cookieValue)
	successiveFailureCountMutex.Unlock()
}

//ResetSuccessiveFailureMap make map again
func ResetSuccessiveFailureMap() {
	successiveFailureCountMutex.Lock()
	successiveFailureCount = make(map[string]int)
	successiveFailureCountMutex.Unlock()
}

//IncreaseSuccessiveFailureCount increase failure count
func IncreaseSuccessiveFailureCount(cookieValue string) {
	successiveFailureCountMutex.Lock()
	c, ok := successiveFailureCount[cookieValue]
	if ok {
		successiveFailureCount[cookieValue] = c + 1
		successiveFailureCountMutex.Unlock()
		return
	}
	successiveFailureCount[cookieValue] = 1
	successiveFailureCountMutex.Unlock()
	return
}

//GetSuccessiveFailureCount get failure count
func GetSuccessiveFailureCount(cookieValue string) int {
	successiveFailureCountMutex.RLock()
	defer successiveFailureCountMutex.RUnlock()
	return successiveFailureCount[cookieValue]
}

//SessionStickinessStrategy is strategy
type SessionStickinessStrategy struct {
	instances []*registry.MicroServiceInstance
	mtx       sync.Mutex
	sessionID string
}

func newSessionStickinessStrategy() Strategy {
	return &SessionStickinessStrategy{}
}

// ReceiveData receive data
func (r *SessionStickinessStrategy) ReceiveData(inv *invocation.Invocation, instances []*registry.MicroServiceInstance, serviceName string) {
	r.instances = instances
	r.sessionID = session.GetSessionID(getNamespace(inv))
}
func getNamespace(i *invocation.Invocation) string {
	if metadata, ok := i.Metadata[common.SessionNameSpaceKey]; ok {
		if v, ok := metadata.(string); ok {
			return v
		}
	}
	return common.SessionNameSpaceDefaultValue
}

// Pick return instance
func (r *SessionStickinessStrategy) Pick() (*registry.MicroServiceInstance, error) {
	instanceAddr, ok := session.Get(r.sessionID)
	if ok {
		if len(r.instances) == 0 {
			return nil, ErrNoneAvailableInstance
		}

		for _, instance := range r.instances {
			if instanceAddr == instance.EndpointsMap[instance.DefaultProtocol] {
				return instance, nil
			}
		}
		// if micro service instance goes down then related entry in endpoint map will be deleted,
		//so instead of sending nil, a new instance can be selected using round robin
		return r.pick()
	}
	return r.pick()

}
func (r *SessionStickinessStrategy) pick() (*registry.MicroServiceInstance, error) {
	if len(r.instances) == 0 {
		return nil, ErrNoneAvailableInstance
	}

	r.mtx.Lock()
	instance := r.instances[i%len(r.instances)]
	i++
	r.mtx.Unlock()

	return instance, nil
}
