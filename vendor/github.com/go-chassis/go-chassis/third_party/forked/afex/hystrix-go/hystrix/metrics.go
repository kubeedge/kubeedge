package hystrix

import (
	"sync"
	"time"

	"github.com/go-chassis/go-chassis/third_party/forked/afex/hystrix-go/hystrix/metric_collector"
	"github.com/go-chassis/go-chassis/third_party/forked/afex/hystrix-go/hystrix/rolling"
	"github.com/go-mesh/openlogging"
)

type commandExecution struct {
	Types       []string      `json:"types"`
	Start       time.Time     `json:"start_time"`
	RunDuration time.Duration `json:"run_duration"`
}

type metricExchange struct {
	Name    string
	Updates chan *commandExecution
	Mutex   *sync.RWMutex

	metricCollectors []metricCollector.MetricCollector
}

func newMetricExchange(name string, num int) *metricExchange {
	m := &metricExchange{}
	m.Name = name

	m.Updates = make(chan *commandExecution, 2000)
	m.Mutex = &sync.RWMutex{}
	m.metricCollectors = metricCollector.Registry.InitializeMetricCollectors(name)
	m.Reset()
	for i := 0; i < num; i++ {
		go m.Monitor()
	}
	openlogging.GetLogger().Debugf(" launched [%d] Metrics consumer", num)
	return m
}

// The Default Collector function will panic if collectors are not setup to specification.
func (m *metricExchange) DefaultCollector() *metricCollector.DefaultMetricCollector {
	if len(m.metricCollectors) < 1 {
		panic("No Metric Collectors Registered")
	}
	collection, ok := m.metricCollectors[0].(*metricCollector.DefaultMetricCollector)
	if !ok {
		panic("Default metric collector is not registered correctly. The default metric collector must be registered first")
	}
	return collection
}

func (m *metricExchange) Monitor() {
	for update := range m.Updates {
		// we only grab a read lock to make sure Reset() isn't changing the numbers.
		m.Mutex.RLock()

		totalDuration := time.Since(update.Start)
		for _, collector := range m.metricCollectors {
			m.IncrementMetrics(collector, update, totalDuration)
		}

		m.Mutex.RUnlock()
	}
}

func (m *metricExchange) IncrementMetrics(collector metricCollector.MetricCollector, update *commandExecution, totalDuration time.Duration) {
	// granular Metrics
	if update.Types[0] == "success" {
		collector.IncrementAttempts()
		collector.IncrementSuccesses()
	}
	if update.Types[0] == "failure" {
		collector.IncrementFailures()

		collector.IncrementAttempts()
		collector.IncrementErrors()
	}
	if update.Types[0] == "rejected" {
		collector.IncrementRejects()

		collector.IncrementAttempts()
		collector.IncrementErrors()
	}
	if update.Types[0] == "short-circuit" {
		collector.IncrementShortCircuits()

		collector.IncrementAttempts()
	}
	if update.Types[0] == "timeout" {
		collector.IncrementTimeouts()

		collector.IncrementAttempts()
		collector.IncrementErrors()
	}

	if len(update.Types) > 1 {
		// fallback Metrics
		if update.Types[1] == "fallback-success" {
			collector.IncrementFallbackSuccesses()
		}
		if update.Types[1] == "fallback-failure" {
			collector.IncrementFallbackFailures()
		}
	}

	collector.UpdateTotalDuration(totalDuration)
	collector.UpdateRunDuration(update.RunDuration)

}

func (m *metricExchange) Reset() {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	for _, collector := range m.metricCollectors {
		collector.Reset()
	}
}

func (m *metricExchange) Requests() *rolling.Number {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()
	return m.requestsLocked()
}

func (m *metricExchange) requestsLocked() *rolling.Number {
	return m.DefaultCollector().NumRequests()
}

func (m *metricExchange) ErrorPercent(now time.Time) int {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	var errPct float64
	reqs := m.requestsLocked().Sum(now)
	errs := m.DefaultCollector().Errors().Sum(now)

	if reqs > 0 {
		errPct = (float64(errs) / float64(reqs)) * 100
	}

	return int(errPct + 0.5)
}

func (m *metricExchange) IsHealthy(now time.Time) bool {
	return m.ErrorPercent(now) < getSettings(m.Name).ErrorPercentThreshold
}
