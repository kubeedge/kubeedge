/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "konnectivity_network_proxy"
	subsystem = "server"

	// Proxy is the ProxyService method used to handle incoming streams.
	Proxy = "Proxy"

	// Connect is the AgentService method used to establish next hop.
	Connect = "Connect"
)

var (
	// Use buckets ranging from 10 ns to 12.5 seconds.
	latencyBuckets = []float64{0.000001, 0.00001, 0.0001, 0.005, 0.025, 0.1, 0.5, 2.5, 12.5}

	// Metrics provides access to all dial metrics.
	Metrics = newServerMetrics()
)

// ServerMetrics includes all the metrics of the proxy server.
type ServerMetrics struct {
	latencies         *prometheus.HistogramVec
	frontendLatencies *prometheus.HistogramVec
	connections       *prometheus.GaugeVec
	httpConnections   prometheus.Gauge
	backend           *prometheus.GaugeVec
	pendingDials      *prometheus.GaugeVec
}

// newServerMetrics create a new ServerMetrics, configured with default metric names.
func newServerMetrics() *ServerMetrics {
	latencies := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "dial_duration_seconds",
			Help:      "Latency of dial to the remote endpoint in seconds",
			Buckets:   latencyBuckets,
		},
		[]string{},
	)
	frontendLatencies := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "frontend_write_duration_seconds",
			Help:      "Latency of write to the frontend in seconds",
			Buckets:   latencyBuckets,
		},
		[]string{},
	)
	connections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "grpc_connections",
			Help:      "Number of current grpc connections, partitioned by service method.",
		},
		[]string{
			"service_method",
		},
	)
	httpConnections := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_connections",
			Help:      "Number of current HTTP CONNECT connections",
		},
	)
	backend := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "ready_backend_connections",
			Help:      "Number of konnectivity agent connected to the proxy server",
		},
		[]string{},
	)
	pendingDials := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "pending_backend_dials",
			Help:      "Current number of pending backend dial requests",
		},
		[]string{},
	)

	prometheus.MustRegister(latencies)
	prometheus.MustRegister(frontendLatencies)
	prometheus.MustRegister(connections)
	prometheus.MustRegister(httpConnections)
	prometheus.MustRegister(backend)
	prometheus.MustRegister(pendingDials)
	return &ServerMetrics{
		latencies:         latencies,
		frontendLatencies: frontendLatencies,
		connections:       connections,
		httpConnections:   httpConnections,
		backend:           backend,
		pendingDials:      pendingDials,
	}
}

// Reset resets the metrics.
func (a *ServerMetrics) Reset() {
	a.latencies.Reset()
	a.frontendLatencies.Reset()
}

// ObserveDialLatency records the latency of dial to the remote endpoint.
func (a *ServerMetrics) ObserveDialLatency(elapsed time.Duration) {
	a.latencies.WithLabelValues().Observe(elapsed.Seconds())
}

// ObserveFrontendWriteLatency records the latency of dial to the remote endpoint.
func (a *ServerMetrics) ObserveFrontendWriteLatency(elapsed time.Duration) {
	a.frontendLatencies.WithLabelValues().Observe(elapsed.Seconds())
}

// ConnectionInc increments a new grpc client connection.
func (a *ServerMetrics) ConnectionInc(serviceMethod string) {
	a.connections.With(prometheus.Labels{"service_method": serviceMethod}).Inc()
}

// ConnectionDec decrements a finished grpc client connection.
func (a *ServerMetrics) ConnectionDec(serviceMethod string) {
	a.connections.With(prometheus.Labels{"service_method": serviceMethod}).Dec()
}

// HTTPConnectionDec increments a new HTTP CONNECTION connection.
func (a *ServerMetrics) HTTPConnectionInc() { a.httpConnections.Inc() }

// HTTPConnectionDec decrements a finished HTTP CONNECTION connection.
func (a *ServerMetrics) HTTPConnectionDec() { a.httpConnections.Dec() }

// SetBackendCount sets the number of backend connection.
func (a *ServerMetrics) SetBackendCount(count int) {
	a.backend.WithLabelValues().Set(float64(count))
}

// SetPendingDialCount sets the number of pending dials.
func (a *ServerMetrics) SetPendingDialCount(count int) {
	a.pendingDials.WithLabelValues().Set(float64(count))
}
