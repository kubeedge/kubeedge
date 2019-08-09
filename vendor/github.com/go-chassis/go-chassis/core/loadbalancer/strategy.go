package loadbalancer

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-mesh/openlogging"
)

var strategies = make(map[string]func() Strategy)
var i int

func init() {
	rand.Seed(time.Now().UnixNano())
	rand.Seed(time.Now().Unix())
	i = rand.Int()
}

// InstallStrategy install strategy
func InstallStrategy(name string, s func() Strategy) {
	strategies[name] = s
	openlogging.GetLogger().Debugf("Installed strategy plugin: %s.", name)
}

// GetStrategyPlugin get strategy plugin
func GetStrategyPlugin(name string) (func() Strategy, error) {
	s, ok := strategies[name]
	if !ok {
		return nil, fmt.Errorf("don't support strategyName [%s]", name)
	}

	return s, nil
}
