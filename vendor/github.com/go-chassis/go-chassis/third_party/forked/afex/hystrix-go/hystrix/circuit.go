package hystrix

// Forked from github.com/afex/hystrix-go/hystrix
// Some parts of this file have been modified to make it functional in this package
import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mesh/openlogging"
)

// CircuitBreaker is created for each ExecutorPool to track whether requests
// should be attempted, or rejected if the Health of the circuit is too low.
type CircuitBreaker struct {
	Name                   string
	open                   bool
	enabled                bool
	forceOpen              bool
	forceClosed            bool
	mutex                  *sync.RWMutex
	openedOrLastTestedTime int64
	executorPool           *executorPool
	metrics                *metricExchange
}

var (
	// ErrCBNotExist occurs when no CircuitBreaker exists
	ErrCBNotExist = errors.New("circuit breaker not exist")
)

var (
	circuitBreakersMutex *sync.RWMutex
	circuitBreakers      map[string]*CircuitBreaker
)

func init() {
	circuitBreakersMutex = &sync.RWMutex{}
	circuitBreakers = make(map[string]*CircuitBreaker)
}

// IsCircuitBreakerOpen returns whether a circuitBreaker is open for an interface
func IsCircuitBreakerOpen(name string) (bool, error) {
	circuitBreakersMutex.Lock()
	defer circuitBreakersMutex.Unlock()
	if c, ok := circuitBreakers[name]; ok {
		return c.IsOpen(), nil
	} else {
		return false, ErrCBNotExist
	}
}

// GetCircuit returns the circuit for the given command and whether this call created it.
func GetCircuit(name string) (*CircuitBreaker, bool, error) {
	circuitBreakersMutex.RLock()
	_, ok := circuitBreakers[name]
	if !ok {
		circuitBreakersMutex.RUnlock()
		circuitBreakersMutex.Lock()
		defer circuitBreakersMutex.Unlock()
		// because we released the rlock before we obtained the exclusive lock,
		// we need to double check that some other thread didn't beat us to
		// creation.
		if cb, ok := circuitBreakers[name]; ok {
			return cb, false, nil
		}
		openlogging.GetLogger().Infof("new circuit [%s] is protecting your service", name)
		circuitBreakers[name] = newCircuitBreaker(name)
	} else {
		defer circuitBreakersMutex.RUnlock()
	}

	return circuitBreakers[name], !ok, nil
}
func FlushByName(name string) {
	circuitBreakersMutex.Lock()
	defer circuitBreakersMutex.Unlock()
	cb, ok := circuitBreakers[name]
	if ok {
		log.Println("Delete Circuit Breaker:", name)
		cb.metrics.Reset()
		cb.executorPool.Metrics.Reset()
		delete(circuitBreakers, name)
	}

}

// Flush purges all circuit and metric information from memory.
func Flush() {
	circuitBreakersMutex.Lock()
	defer circuitBreakersMutex.Unlock()

	for name, cb := range circuitBreakers {
		cb.metrics.Reset()
		cb.executorPool.Metrics.Reset()
		delete(circuitBreakers, name)
	}
}

// newCircuitBreaker creates a CircuitBreaker with associated Health
func newCircuitBreaker(name string) *CircuitBreaker {
	c := &CircuitBreaker{}
	c.Name = name
	c.metrics = newMetricExchange(name, getSettings(name).MetricsConsumerNum)
	c.executorPool = newExecutorPool(name)
	c.mutex = &sync.RWMutex{}
	//定制治理选项forceClosed
	c.forceOpen = getSettings(name).ForceOpen
	c.forceClosed = getSettings(name).ForceClose
	c.enabled = getSettings(name).CircuitBreakerEnabled
	return c
}

// toggleForceOpen allows manually causing the fallback logic for all instances
// of a given command.
func (circuit *CircuitBreaker) ToggleForceOpen(toggle bool) error {
	circuit, _, err := GetCircuit(circuit.Name)
	if err != nil {
		return err
	}

	circuit.forceOpen = toggle
	return nil
}

// IsOpen is called before any Command execution to check whether or
// not it should be attempted. An "open" circuit means it is disabled.
func (circuit *CircuitBreaker) IsOpen() bool {
	circuit.mutex.RLock()
	o := circuit.forceOpen || circuit.open
	circuit.mutex.RUnlock()

	if o {
		return true
	}

	if uint64(circuit.metrics.Requests().Sum(time.Now())) < getSettings(circuit.Name).RequestVolumeThreshold {
		return false
	}

	if !circuit.metrics.IsHealthy(time.Now()) {
		// too many failures, open the circuit
		circuit.setOpen()
		return true
	}

	return false
}

// AllowRequest is checked before a command executes, ensuring that circuit state and metric health allow it.
// When the circuit is open, this call will occasionally return true to measure whether the external service
// has recovered.
func (circuit *CircuitBreaker) AllowRequest() bool {
	if circuit.forceOpen {
		return false
	}
	//如果不允许熔断，直接返回
	if circuit.forceClosed {
		return true
	}
	return !circuit.IsOpen() || circuit.allowSingleTest()
}

func (circuit *CircuitBreaker) allowSingleTest() bool {
	circuit.mutex.RLock()
	defer circuit.mutex.RUnlock()

	now := time.Now().UnixNano()
	openedOrLastTestedTime := atomic.LoadInt64(&circuit.openedOrLastTestedTime)
	if circuit.open && now > openedOrLastTestedTime+getSettings(circuit.Name).SleepWindow.Nanoseconds() {
		swapped := atomic.CompareAndSwapInt64(&circuit.openedOrLastTestedTime, openedOrLastTestedTime, now)
		if swapped {
			openlogging.GetLogger().Warnf("hystrix-go: allowing single test to possibly close circuit %v", circuit.Name)
		}
		return swapped
	}

	return false
}

func (circuit *CircuitBreaker) setOpen() {

	circuit.mutex.Lock()
	defer circuit.mutex.Unlock()

	if circuit.open {
		return
	}

	openlogging.GetLogger().Infof("hystrix-go: opening circuit %v", circuit.Name)

	circuit.openedOrLastTestedTime = time.Now().UnixNano()
	circuit.open = true
}

func (circuit *CircuitBreaker) setClose() {
	circuit.mutex.Lock()
	defer circuit.mutex.Unlock()

	if !circuit.open {
		return
	}

	openlogging.GetLogger().Warnf("hystrix-go: closing circuit %v", circuit.Name)

	circuit.open = false
	circuit.metrics.Reset()
}

// ReportEvent records command metrics for tracking recent error rates and exposing data to the dashboard.
func (circuit *CircuitBreaker) ReportEvent(eventTypes []string, start time.Time, runDuration time.Duration) error {
	if len(eventTypes) == 0 {
		return fmt.Errorf("no event types sent for metrics")
	}

	if eventTypes[0] == "success" && circuit.open {
		circuit.setClose()
	}

	select {
	case circuit.metrics.Updates <- &commandExecution{
		Types:       eventTypes,
		Start:       start,
		RunDuration: runDuration,
	}:
	default:
		return CircuitError{Message: fmt.Sprintf("metrics channel (%v) is at capacity", circuit.Name)}
	}

	return nil
}
