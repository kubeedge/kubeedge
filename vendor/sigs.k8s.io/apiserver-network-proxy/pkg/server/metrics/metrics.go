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
	// Use buckets ranging from 5 ms to 12.5 seconds.
	latencyBuckets = []float64{0.005, 0.025, 0.1, 0.5, 2.5, 12.5}

	// Metrics provides access to all dial metrics.
	Metrics = newServerMetrics()
)

// ServerMetrics includes all the metrics of the proxy server.
type ServerMetrics struct {
	latencies *prometheus.HistogramVec
	connections *prometheus.GaugeVec
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
	
	prometheus.MustRegister(latencies)
	prometheus.MustRegister(connections)
	return &ServerMetrics{latencies: latencies, connections: connections}
}

// Reset resets the metrics.
func (a *ServerMetrics) Reset() {
	a.latencies.Reset()
}

// ObserveDialLatency records the latency of dial to the remote endpoint.
func (a *ServerMetrics) ObserveDialLatency(elapsed time.Duration) {
	a.latencies.WithLabelValues().Observe(elapsed.Seconds())
}

// ConnectionInc increments a new grpc client connection.
func (a *ServerMetrics) ConnectionInc(service_method string) {
	a.connections.With(prometheus.Labels{"service_method": service_method}).Inc()
}

// ConnectionDec decrements a finished grpc client connection.
func (a *ServerMetrics) ConnectionDec(service_method string) {
	a.connections.With(prometheus.Labels{"service_method": service_method}).Dec()
}
