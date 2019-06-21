package metricCollector

import (
	"sync"
	"time"
)

// Registry is the default metricCollectorRegistry that circuits will use to
// collect statistics about the health of the circuit.
var Registry = metricCollectorRegistry{
	lock: &sync.RWMutex{},
	registry: []func(name string) MetricCollector{
		newDefaultMetricCollector,
	},
}

type metricCollectorRegistry struct {
	lock     *sync.RWMutex
	registry []func(name string) MetricCollector
}

// InitializeMetricCollectors runs the registried MetricCollector Initializers to create an array of MetricCollectors.
func (m *metricCollectorRegistry) InitializeMetricCollectors(name string) []MetricCollector {
	m.lock.RLock()
	defer m.lock.RUnlock()

	metrics := make([]MetricCollector, len(m.registry))
	for i, metricCollectorInitializer := range m.registry {
		metrics[i] = metricCollectorInitializer(name)
	}
	return metrics
}

// Register places a MetricCollector Initializer in the registry maintained by this metricCollectorRegistry.
func (m *metricCollectorRegistry) Register(initMetricCollector func(string) MetricCollector) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.registry = append(m.registry, initMetricCollector)
}

// MetricCollector represents the contract that all collectors must fulfill to gather circuit statistics.
// Implementations of this interface do not have to maintain locking around thier data stores so long as
// they are not modified outside of the hystrix context.
type MetricCollector interface {
	// IncrementAttempts increments the number of updates.
	IncrementAttempts()
	// IncrementErrors increments the number of unsuccessful attempts.
	// Attempts minus Errors will equal successes within a time range.
	// Errors are any result from an attempt that is not a success.
	IncrementErrors()
	// IncrementSuccesses increments the number of requests that succeed.
	IncrementSuccesses()
	// IncrementFailures increments the number of requests that fail.
	IncrementFailures()
	// IncrementRejects increments the number of requests that are rejected.
	IncrementRejects()
	// IncrementShortCircuits increments the number of requests that short circuited due to the circuit being open.
	IncrementShortCircuits()
	// IncrementTimeouts increments the number of timeouts that occurred in the circuit breaker.
	IncrementTimeouts()
	// IncrementFallbackSuccesses increments the number of successes that occurred during the execution of the fallback function.
	IncrementFallbackSuccesses()
	// IncrementFallbackFailures increments the number of failures that occurred during the execution of the fallback function.
	IncrementFallbackFailures()
	// UpdateTotalDuration updates the internal counter of how long we've run for.
	UpdateTotalDuration(timeSinceStart time.Duration)
	// UpdateRunDuration updates the internal counter of how long the last run took.
	UpdateRunDuration(runDuration time.Duration)
	// Reset resets the internal counters and timers.
	Reset()
}
