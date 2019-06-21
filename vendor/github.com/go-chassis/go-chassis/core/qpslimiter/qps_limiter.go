package qpslimiter

import (
	"strconv"
	"sync"

	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/core/lager"

	"go.uber.org/ratelimit"
)

// constant qps default rate
const (
	DefaultRate = 2147483647
)

// QPSLimiterMap qps limiter map struct
type QPSLimiterMap struct {
	KeyMap map[string]ratelimit.Limiter
	sync.RWMutex
}

// variables of qps limiter ansd mutex variable
var (
	qpsLimiter *QPSLimiterMap
	once       = new(sync.Once)
)

// GetQPSTrafficLimiter get qps traffic limiter
func GetQPSTrafficLimiter() *QPSLimiterMap {
	initializeMap := func() {
		qpsLimiter = &QPSLimiterMap{}
		qpsLimiter.KeyMap = make(map[string]ratelimit.Limiter)
	}

	once.Do(initializeMap)
	return qpsLimiter
}

// ProcessQPSTokenReq process qps token request
func (qpsL *QPSLimiterMap) ProcessQPSTokenReq(key string, qpsRate int) {
	qpsL.RLock()

	limiter, ok := qpsL.KeyMap[key]
	if !ok {
		qpsL.RUnlock()
		//If the key operation is not present in the map, then add the new key operation to the map
		qpsL.ProcessDefaultRateRpsTokenReq(key, qpsRate)
		return
	}

	qpsL.RUnlock()
	limiter.Take()

	return
}

// ProcessDefaultRateRpsTokenReq process default rate pps token request
func (qpsL *QPSLimiterMap) ProcessDefaultRateRpsTokenReq(key string, qpsRate int) {
	var bucketSize int

	// add a limiter object for the newly found operation in the Default Hash map
	// so that the default rate will be applied to subsequent token requests to this new operation
	if qpsRate >= 1 {
		bucketSize = int(qpsRate)
	} else {
		bucketSize = DefaultRate
	}

	qpsL.Lock()
	// Create a new bucket for the new operation
	r := ratelimit.New(bucketSize)
	qpsL.KeyMap[key] = r
	qpsL.Unlock()

	r.Take()

	return
}

// GetQPSRate get qps rate
func GetQPSRate(rateConfig string) (int, bool) {
	qpsRate := archaius.GetInt(rateConfig, DefaultRate)
	if qpsRate == DefaultRate {
		return qpsRate, false
	}

	return qpsRate, true
}

// GetQPSRateWithPriority get qps rate with priority
func (qpsL *QPSLimiterMap) GetQPSRateWithPriority(cmd ...string) (int, string) {
	var (
		qpsVal      int
		configExist bool
	)
	for _, c := range cmd {
		qpsVal, configExist = GetQPSRate(c)
		if configExist {
			return qpsVal, c
		}
	}

	return DefaultRate, cmd[len(cmd)-1]

}

// UpdateRateLimit update rate limit
func (qpsL *QPSLimiterMap) UpdateRateLimit(key string, value interface{}) {
	switch v := value.(type) {
	case int:
		qpsL.ProcessDefaultRateRpsTokenReq(key, value.(int))
	case string:
		convertedIntValue, err := strconv.Atoi(value.(string))
		if err != nil {
			lager.Logger.Warnf("Invalid Value type received for QPSLateLimiter: %v", v, err)
		} else {
			qpsL.ProcessDefaultRateRpsTokenReq(key, convertedIntValue)
		}
	default:
		lager.Logger.Warnf("Invalid Value type received for QPSLateLimiter: %v", v)
	}
}

// DeleteRateLimiter delete rate limiter
func (qpsL *QPSLimiterMap) DeleteRateLimiter(key string) {
	qpsL.Lock()
	delete(qpsL.KeyMap, key)
	qpsL.Unlock()
}
