/*
Copyright 2017 The Kubernetes Authors.

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

	commonmetrics "sigs.k8s.io/apiserver-network-proxy/konnectivity-client/pkg/common/metrics"
	"sigs.k8s.io/apiserver-network-proxy/konnectivity-client/proto/client"
)

type Direction string

const (
	Namespace = "konnectivity_network_proxy"
	Subsystem = "agent"

	// DirectionToServer indicates that the agent attempts to send a packet
	// to the proxy server.
	DirectionToServer Direction = "to_server"
	// DirectionFromServer indicates that the agent attempts to receive a
	// packet from the proxy server.
	DirectionFromServer Direction = "from_server"
)

var (
	// Use buckets ranging from 5 ms to 30 seconds.
	latencyBuckets = []float64{0.005, 0.025, 0.1, 0.5, 2.5, 10, 30}

	// Metrics provides access to all dial metrics.
	Metrics = newAgentMetrics()
)

// AgentMetrics includes all the metrics of the proxy agent.
type AgentMetrics struct {
	dialLatencies       *prometheus.HistogramVec
	serverFailures      *prometheus.CounterVec
	dialFailures        *prometheus.CounterVec
	serverConnections   *prometheus.GaugeVec
	endpointConnections *prometheus.GaugeVec
	streamPackets       *prometheus.CounterVec
	streamErrors        *prometheus.CounterVec
}

// newAgentMetrics create a new AgentMetrics, configured with default metric names.
func newAgentMetrics() *AgentMetrics {
	dialLatencies := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "dial_duration_seconds",
			Help:      "Latency of dial to the remote endpoint in seconds",
			Buckets:   latencyBuckets,
		},
		[]string{},
	)
	serverFailures := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "server_connection_failure_count",
			Help:      "Count of failures to send to or receive from the proxy server, labeled by the direction (from_server or to_server). DEPRECATED, please use stream_events_error_total",
		},
		[]string{"direction"},
	)
	dialFailures := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "endpoint_dial_failure_total",
			Help:      "Number of failures dialing the remote endpoint, by reason (example: timeout).",
		},
		[]string{"reason"},
	)
	serverConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "open_server_connections",
			Help:      "Current number of open server connections.",
		},
		[]string{},
	)
	endpointConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "open_endpoint_connections",
			Help:      "Current number of open endpoint connections.",
		},
		[]string{},
	)
	streamPackets := commonmetrics.MakeStreamPacketsTotalMetric(Namespace, Subsystem)
	streamErrors := commonmetrics.MakeStreamErrorsTotalMetric(Namespace, Subsystem)
	prometheus.MustRegister(dialLatencies)
	prometheus.MustRegister(serverFailures)
	prometheus.MustRegister(dialFailures)
	prometheus.MustRegister(serverConnections)
	prometheus.MustRegister(endpointConnections)
	prometheus.MustRegister(streamPackets)
	prometheus.MustRegister(streamErrors)
	return &AgentMetrics{
		dialLatencies:       dialLatencies,
		serverFailures:      serverFailures,
		dialFailures:        dialFailures,
		serverConnections:   serverConnections,
		endpointConnections: endpointConnections,
		streamPackets:       streamPackets,
		streamErrors:        streamErrors,
	}

}

// Reset resets the metrics.
func (a *AgentMetrics) Reset() {
	a.dialLatencies.Reset()
	a.serverFailures.Reset()
	a.dialFailures.Reset()
	a.serverConnections.Reset()
	a.endpointConnections.Reset()
	a.streamPackets.Reset()
	a.streamErrors.Reset()
}

// ObserveServerFailure records a failure to send to or receive from the proxy
// server, labeled by the direction.
func (a *AgentMetrics) ObserveServerFailureDeprecated(direction Direction) {
	a.serverFailures.WithLabelValues(string(direction)).Inc()
}

type DialFailureReason string

const (
	DialFailureTimeout DialFailureReason = "timeout"
	DialFailureUnknown DialFailureReason = "unknown"
)

// ObserveDialLatency records the latency of dial to the remote endpoint.
func (a *AgentMetrics) ObserveDialLatency(elapsed time.Duration) {
	a.dialLatencies.WithLabelValues().Observe(elapsed.Seconds())
}

// ObserveDialFailure records a remote endpoint dial failure.
func (a *AgentMetrics) ObserveDialFailure(reason DialFailureReason) {
	a.dialFailures.WithLabelValues(string(reason)).Inc()
}

func (a *AgentMetrics) SetServerConnectionsCount(count int) {
	a.serverConnections.WithLabelValues().Set(float64(count))
}

// EndpointConnectionInc increments a new endpoint connection.
func (a *AgentMetrics) EndpointConnectionInc() {
	a.endpointConnections.WithLabelValues().Inc()
}

// EndpointConnectionDec decrements a finished endpoint connection.
func (a *AgentMetrics) EndpointConnectionDec() {
	a.endpointConnections.WithLabelValues().Dec()
}

func (a *AgentMetrics) ObservePacket(segment commonmetrics.Segment, packetType client.PacketType) {
	commonmetrics.ObservePacket(a.streamPackets, segment, packetType)
}

func (a *AgentMetrics) ObserveStreamErrorNoPacket(segment commonmetrics.Segment, err error) {
	commonmetrics.ObserveStreamErrorNoPacket(a.streamErrors, segment, err)
}

func (a *AgentMetrics) ObserveStreamError(segment commonmetrics.Segment, err error, packetType client.PacketType) {
	commonmetrics.ObserveStreamError(a.streamErrors, segment, err, packetType)
}
