package hystrix

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chassis/go-chassis/third_party/forked/afex/hystrix-go/hystrix/rolling"
	"github.com/go-mesh/openlogging"
)

const (
	streamEventBufferSize = 10
)

// NewStreamHandler returns a server capable of exposing dashboard metrics via HTTP.
func NewStreamHandler() *StreamHandler {
	return &StreamHandler{}
}

// StreamHandler publishes metrics for each command and each pool once a second to all connected HTTP client.
type StreamHandler struct {
	requests map[*http.Request]chan []byte
	mu       sync.RWMutex
	done     chan struct{}
}

// Start begins watching the in-memory circuit breakers for metrics
func (sh *StreamHandler) Start() {
	sh.requests = make(map[*http.Request]chan []byte)
	sh.done = make(chan struct{})
	go sh.loop()
}

// Stop shuts down the metric collection routine
func (sh *StreamHandler) Stop() {
	close(sh.done)
}

var _ http.Handler = (*StreamHandler)(nil)

func (sh *StreamHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Make sure that the writer supports flushing.
	f, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	events := sh.register(req)
	defer sh.unregister(req)

	notify := rw.(http.CloseNotifier).CloseNotify()

	rw.Header().Add("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	for {
		select {
		case <-notify:
			// client is gone
			return
		case event := <-events:
			_, err := rw.Write(event)
			if err != nil {
				return
			}
			f.Flush()
		}
	}
}

func (sh *StreamHandler) loop() {
	tick := time.Tick(1 * time.Second)
	for {
		select {
		case <-tick:
			circuitBreakersMutex.RLock()
			for _, cb := range circuitBreakers {
				err := sh.publishMetrics(cb)
				if err != nil {
					openlogging.Warn("publishMetrics err: " + err.Error())
				}
				err = sh.publishThreadPools(cb.executorPool)
				if err != nil {
					openlogging.Warn("publishThreadPools err: " + err.Error())
				}
			}
			circuitBreakersMutex.RUnlock()
		case <-sh.done:
			return
		}
	}
}

func (sh *StreamHandler) publishMetrics(cb *CircuitBreaker) error {
	now := time.Now()
	reqCount := cb.metrics.Requests().Sum(now)
	errCount := cb.metrics.DefaultCollector().Errors().Sum(now)
	errPct := cb.metrics.ErrorPercent(now)

	eventBytes, err := json.Marshal(&streamCmdMetric{
		Type:           "HystrixCommand",
		Name:           cb.Name,
		Group:          cb.Name,
		Time:           currentTime(),
		ReportingHosts: 1,

		RequestCount:       uint32(reqCount),
		ErrorCount:         uint32(errCount),
		ErrorPct:           uint32(errPct),
		CircuitBreakerOpen: cb.IsOpen(),

		RollingCountSuccess:            uint32(cb.metrics.DefaultCollector().Successes().Sum(now)),
		RollingCountFailure:            uint32(cb.metrics.DefaultCollector().Failures().Sum(now)),
		RollingCountThreadPoolRejected: uint32(cb.metrics.DefaultCollector().Rejects().Sum(now)),
		RollingCountShortCircuited:     uint32(cb.metrics.DefaultCollector().ShortCircuits().Sum(now)),
		RollingCountTimeout:            uint32(cb.metrics.DefaultCollector().Timeouts().Sum(now)),
		RollingCountFallbackSuccess:    uint32(cb.metrics.DefaultCollector().FallbackSuccesses().Sum(now)),
		RollingCountFallbackFailure:    uint32(cb.metrics.DefaultCollector().FallbackFailures().Sum(now)),

		LatencyTotal:       generateLatencyTimings(cb.metrics.DefaultCollector().TotalDuration()),
		LatencyTotalMean:   cb.metrics.DefaultCollector().TotalDuration().Mean(),
		LatencyExecute:     generateLatencyTimings(cb.metrics.DefaultCollector().RunDuration()),
		LatencyExecuteMean: cb.metrics.DefaultCollector().RunDuration().Mean(),

		// TODO: all hard-coded values should become configurable settings, per circuit

		RollingStatsWindow:         10000,
		ExecutionIsolationStrategy: "THREAD",

		CircuitBreakerEnabled:                true,
		CircuitBreakerForceClosed:            false,
		CircuitBreakerForceOpen:              cb.forceOpen,
		CircuitBreakerErrorThresholdPercent:  uint32(getSettings(cb.Name).ErrorPercentThreshold),
		CircuitBreakerSleepWindow:            uint32(getSettings(cb.Name).SleepWindow.Seconds() * 1000),
		CircuitBreakerRequestVolumeThreshold: uint32(getSettings(cb.Name).RequestVolumeThreshold),
	})
	if err != nil {
		return err
	}
	err = sh.writeToRequests(eventBytes)
	if err != nil {
		return err
	}

	return nil
}

func (sh *StreamHandler) publishThreadPools(pool *executorPool) error {
	now := time.Now()

	eventBytes, err := json.Marshal(&streamThreadPoolMetric{
		Type:           "HystrixThreadPool",
		Name:           pool.Name,
		ReportingHosts: 1,

		CurrentActiveCount:        uint32(pool.ActiveCount()),
		CurrentTaskCount:          0,
		CurrentCompletedTaskCount: 0,

		RollingCountThreadsExecuted: uint32(pool.Metrics.Executed.Sum(now)),
		RollingMaxActiveThreads:     uint32(pool.Metrics.MaxActiveRequests.Max(now)),

		CurrentPoolSize:        uint32(pool.Max),
		CurrentCorePoolSize:    uint32(pool.Max),
		CurrentLargestPoolSize: uint32(pool.Max),
		CurrentMaximumPoolSize: uint32(pool.Max),

		RollingStatsWindow:          10000,
		QueueSizeRejectionThreshold: 0,
		CurrentQueueSize:            0,
	})
	if err != nil {
		return err
	}
	err = sh.writeToRequests(eventBytes)
	if err != nil {
		openlogging.Warn("write err: " + err.Error())
	}
	return nil
}

func (sh *StreamHandler) writeToRequests(eventBytes []byte) error {
	var b bytes.Buffer
	_, err := b.Write([]byte("data:"))
	if err != nil {
		return err
	}

	_, err = b.Write(eventBytes)
	if err != nil {
		return err
	}
	_, err = b.Write([]byte("\n\n"))
	if err != nil {
		return err
	}
	dataBytes := b.Bytes()
	sh.mu.RLock()

	for _, requestEvents := range sh.requests {
		select {
		case requestEvents <- dataBytes:
		default:
		}
	}
	sh.mu.RUnlock()

	return nil
}

func (sh *StreamHandler) register(req *http.Request) <-chan []byte {
	sh.mu.RLock()
	events, ok := sh.requests[req]
	sh.mu.RUnlock()
	if ok {
		return events
	}

	events = make(chan []byte, streamEventBufferSize)
	sh.mu.Lock()
	sh.requests[req] = events
	sh.mu.Unlock()
	return events
}

func (sh *StreamHandler) unregister(req *http.Request) {
	sh.mu.Lock()
	delete(sh.requests, req)
	sh.mu.Unlock()
}

func generateLatencyTimings(r *rolling.Timing) streamCmdLatency {
	return streamCmdLatency{
		Timing0:   r.Percentile(0),
		Timing25:  r.Percentile(25),
		Timing50:  r.Percentile(50),
		Timing75:  r.Percentile(75),
		Timing90:  r.Percentile(90),
		Timing95:  r.Percentile(95),
		Timing99:  r.Percentile(99),
		Timing995: r.Percentile(99.5),
		Timing100: r.Percentile(100),
	}
}

type streamCmdMetric struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	Group          string `json:"group"`
	Time           int64  `json:"currentTime"`
	ReportingHosts uint32 `json:"reportingHosts"`

	// Health
	RequestCount       uint32 `json:"requestCount"`
	ErrorCount         uint32 `json:"errorCount"`
	ErrorPct           uint32 `json:"errorPercentage"`
	CircuitBreakerOpen bool   `json:"isCircuitBreakerOpen"`

	RollingCountCollapsedRequests  uint32 `json:"rollingCountCollapsedRequests"`
	RollingCountExceptionsThrown   uint32 `json:"rollingCountExceptionsThrown"`
	RollingCountFailure            uint32 `json:"rollingCountFailure"`
	RollingCountFallbackFailure    uint32 `json:"rollingCountFallbackFailure"`
	RollingCountFallbackRejection  uint32 `json:"rollingCountFallbackRejection"`
	RollingCountFallbackSuccess    uint32 `json:"rollingCountFallbackSuccess"`
	RollingCountResponsesFromCache uint32 `json:"rollingCountResponsesFromCache"`
	RollingCountSemaphoreRejected  uint32 `json:"rollingCountSemaphoreRejected"`
	RollingCountShortCircuited     uint32 `json:"rollingCountShortCircuited"`
	RollingCountSuccess            uint32 `json:"rollingCountSuccess"`
	RollingCountThreadPoolRejected uint32 `json:"rollingCountThreadPoolRejected"`
	RollingCountTimeout            uint32 `json:"rollingCountTimeout"`

	CurrentConcurrentExecutionCount uint32 `json:"currentConcurrentExecutionCount"`

	LatencyExecuteMean uint32           `json:"latencyExecute_mean"`
	LatencyExecute     streamCmdLatency `json:"latencyExecute"`
	LatencyTotalMean   uint32           `json:"latencyTotal_mean"`
	LatencyTotal       streamCmdLatency `json:"latencyTotal"`

	// Properties
	CircuitBreakerRequestVolumeThreshold             uint32 `json:"propertyValue_circuitBreakerRequestVolumeThreshold"`
	CircuitBreakerSleepWindow                        uint32 `json:"propertyValue_circuitBreakerSleepWindowInMilliseconds"`
	CircuitBreakerErrorThresholdPercent              uint32 `json:"propertyValue_circuitBreakerErrorThresholdPercentage"`
	CircuitBreakerForceOpen                          bool   `json:"propertyValue_circuitBreakerForceOpen"`
	CircuitBreakerForceClosed                        bool   `json:"propertyValue_circuitBreakerForceClosed"`
	CircuitBreakerEnabled                            bool   `json:"propertyValue_circuitBreakerEnabled"`
	ExecutionIsolationStrategy                       string `json:"propertyValue_executionIsolationStrategy"`
	ExecutionIsolationThreadTimeout                  uint32 `json:"propertyValue_executionIsolationThreadTimeoutInMilliseconds"`
	ExecutionIsolationThreadInterruptOnTimeout       bool   `json:"propertyValue_executionIsolationThreadInterruptOnTimeout"`
	ExecutionIsolationThreadPoolKeyOverride          string `json:"propertyValue_executionIsolationThreadPoolKeyOverride"`
	ExecutionIsolationSemaphoreMaxConcurrentRequests uint32 `json:"propertyValue_executionIsolationSemaphoreMaxConcurrentRequests"`
	FallbackIsolationSemaphoreMaxConcurrentRequests  uint32 `json:"propertyValue_fallbackIsolationSemaphoreMaxConcurrentRequests"`
	RollingStatsWindow                               uint32 `json:"propertyValue_metricsRollingStatisticalWindowInMilliseconds"`
	RequestCacheEnabled                              bool   `json:"propertyValue_requestCacheEnabled"`
	RequestLogEnabled                                bool   `json:"propertyValue_requestLogEnabled"`
}

type streamCmdLatency struct {
	Timing0   uint32 `json:"0"`
	Timing25  uint32 `json:"25"`
	Timing50  uint32 `json:"50"`
	Timing75  uint32 `json:"75"`
	Timing90  uint32 `json:"90"`
	Timing95  uint32 `json:"95"`
	Timing99  uint32 `json:"99"`
	Timing995 uint32 `json:"99.5"`
	Timing100 uint32 `json:"100"`
}

type streamThreadPoolMetric struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	ReportingHosts uint32 `json:"reportingHosts"`

	CurrentActiveCount        uint32 `json:"currentActiveCount"`
	CurrentCompletedTaskCount uint32 `json:"currentCompletedTaskCount"`
	CurrentCorePoolSize       uint32 `json:"currentCorePoolSize"`
	CurrentLargestPoolSize    uint32 `json:"currentLargestPoolSize"`
	CurrentMaximumPoolSize    uint32 `json:"currentMaximumPoolSize"`
	CurrentPoolSize           uint32 `json:"currentPoolSize"`
	CurrentQueueSize          uint32 `json:"currentQueueSize"`
	CurrentTaskCount          uint32 `json:"currentTaskCount"`

	RollingMaxActiveThreads     uint32 `json:"rollingMaxActiveThreads"`
	RollingCountThreadsExecuted uint32 `json:"rollingCountThreadsExecuted"`

	RollingStatsWindow          uint32 `json:"propertyValue_metricsRollingStatisticalWindowInMilliseconds"`
	QueueSizeRejectionThreshold uint32 `json:"propertyValue_queueSizeRejectionThreshold"`
}

func currentTime() int64 {
	return time.Now().UnixNano() / int64(1000000)
}
