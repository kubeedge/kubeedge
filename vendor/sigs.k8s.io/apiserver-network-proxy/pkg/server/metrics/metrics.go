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

	commonmetrics "sigs.k8s.io/apiserver-network-proxy/konnectivity-client/pkg/common/metrics"
	"sigs.k8s.io/apiserver-network-proxy/konnectivity-client/proto/client"
)

const (
	Namespace = "konnectivity_network_proxy"
	Subsystem = "server"

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
	endpointLatencies *prometheus.HistogramVec
	frontendLatencies *prometheus.HistogramVec
	grpcConnections   *prometheus.GaugeVec
	httpConnections   prometheus.Gauge
	backend           *prometheus.GaugeVec
	pendingDials      *prometheus.GaugeVec
	establishedConns  *prometheus.GaugeVec
	fullRecvChannels  *prometheus.GaugeVec
	dialFailures      *prometheus.CounterVec
	streamPackets     *prometheus.CounterVec
	streamErrors      *prometheus.CounterVec
}

// newServerMetrics create a new ServerMetrics, configured with default metric names.
func newServerMetrics() *ServerMetrics {
	endpointLatencies := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "dial_duration_seconds",
			Help:      "Latency of dial to the remote endpoint in seconds",
			Buckets:   latencyBuckets,
		},
		[]string{},
	)
	frontendLatencies := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "frontend_write_duration_seconds",
			Help:      "Latency of write to the frontend in seconds",
			Buckets:   latencyBuckets,
		},
		[]string{},
	)
	grpcConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "grpc_connections",
			Help:      "Number of current grpc connections, partitioned by service method.",
		},
		[]string{
			"service_method",
		},
	)
	httpConnections := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "http_connections",
			Help:      "Number of current HTTP CONNECT connections",
		},
	)
	backend := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "ready_backend_connections",
			Help:      "Number of konnectivity agent connected to the proxy server",
		},
		[]string{},
	)
	pendingDials := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "pending_backend_dials",
			Help:      "Current number of pending backend dial requests",
		},
		[]string{},
	)
	establishedConns := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "established_connections",
			Help:      "Current number of established end-to-end connections (post-dial).",
		},
		[]string{},
	)
	fullRecvChannels := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "full_receive_channels",
			Help:      "Number of current connections blocked by a full receive channel, partitioned by service method.",
		},
		[]string{
			"service_method",
		},
	)
	dialFailures := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "dial_failure_count",
			Help:      "Number of dial failures observed. Multiple failures can occur for a single dial request.",
		},
		[]string{
			"reason",
		},
	)
	streamPackets := commonmetrics.MakeStreamPacketsTotalMetric(Namespace, Subsystem)
	streamErrors := commonmetrics.MakeStreamErrorsTotalMetric(Namespace, Subsystem)
	prometheus.MustRegister(endpointLatencies)
	prometheus.MustRegister(frontendLatencies)
	prometheus.MustRegister(grpcConnections)
	prometheus.MustRegister(httpConnections)
	prometheus.MustRegister(backend)
	prometheus.MustRegister(pendingDials)
	prometheus.MustRegister(establishedConns)
	prometheus.MustRegister(fullRecvChannels)
	prometheus.MustRegister(dialFailures)
	prometheus.MustRegister(streamPackets)
	prometheus.MustRegister(streamErrors)
	return &ServerMetrics{
		endpointLatencies: endpointLatencies,
		frontendLatencies: frontendLatencies,
		grpcConnections:   grpcConnections,
		httpConnections:   httpConnections,
		backend:           backend,
		pendingDials:      pendingDials,
		establishedConns:  establishedConns,
		fullRecvChannels:  fullRecvChannels,
		dialFailures:      dialFailures,
		streamPackets:     streamPackets,
		streamErrors:      streamErrors,
	}
}

// Reset resets the metrics.
func (s *ServerMetrics) Reset() {
	s.endpointLatencies.Reset()
	s.frontendLatencies.Reset()
	s.grpcConnections.Reset()
	s.backend.Reset()
	s.pendingDials.Reset()
	s.establishedConns.Reset()
	s.fullRecvChannels.Reset()
	s.dialFailures.Reset()
	s.streamPackets.Reset()
	s.streamErrors.Reset()
}

// ObserveDialLatency records the latency of dial to the remote endpoint.
func (s *ServerMetrics) ObserveDialLatency(elapsed time.Duration) {
	s.endpointLatencies.WithLabelValues().Observe(elapsed.Seconds())
}

// ObserveFrontendWriteLatency records the latency of blocking on stream send to the client.
func (s *ServerMetrics) ObserveFrontendWriteLatency(elapsed time.Duration) {
	s.frontendLatencies.WithLabelValues().Observe(elapsed.Seconds())
}

// ConnectionInc increments a new grpc client connection.
func (s *ServerMetrics) ConnectionInc(serviceMethod string) {
	s.grpcConnections.With(prometheus.Labels{"service_method": serviceMethod}).Inc()
}

// ConnectionDec decrements a finished grpc client connection.
func (s *ServerMetrics) ConnectionDec(serviceMethod string) {
	s.grpcConnections.With(prometheus.Labels{"service_method": serviceMethod}).Dec()
}

// HTTPConnectionDec increments a new HTTP CONNECTION connection.
func (s *ServerMetrics) HTTPConnectionInc() { s.httpConnections.Inc() }

// HTTPConnectionDec decrements a finished HTTP CONNECTION connection.
func (s *ServerMetrics) HTTPConnectionDec() { s.httpConnections.Dec() }

// SetBackendCount sets the number of backend connection.
func (s *ServerMetrics) SetBackendCount(count int) {
	s.backend.WithLabelValues().Set(float64(count))
}

// SetPendingDialCount sets the number of pending dials.
func (s *ServerMetrics) SetPendingDialCount(count int) {
	s.pendingDials.WithLabelValues().Set(float64(count))
}

// SetEstablishedConnCount sets the number of established connections.
func (s *ServerMetrics) SetEstablishedConnCount(count int) {
	s.establishedConns.WithLabelValues().Set(float64(count))
}

// FullRecvChannel retrieves the metric for counting full receive channels.
func (s *ServerMetrics) FullRecvChannel(serviceMethod string) prometheus.Gauge {
	return s.fullRecvChannels.With(prometheus.Labels{"service_method": serviceMethod})
}

type DialFailureReason string

const (
	DialFailureNoAgent              DialFailureReason = "no_agent"              // No available agent is connected.
	DialFailureErrorResponse        DialFailureReason = "error_response"        // Dial failure reported by the agent back to the server.
	DialFailureUnrecognizedResponse DialFailureReason = "unrecognized_response" // Dial repsonse received for unrecognozide dial ID.
	DialFailureSendResponse         DialFailureReason = "send_rsp"              // Successful dial response from agent, but failed to send to frontend.
	DialFailureBackendClose         DialFailureReason = "backend_close"         // Received a DIAL_CLS from the backend before the dial completed.
	DialFailureFrontendClose        DialFailureReason = "frontend_close"        // Received a DIAL_CLS from the frontend before the dial completed.
)

func (s *ServerMetrics) ObserveDialFailure(reason DialFailureReason) {
	s.dialFailures.With(prometheus.Labels{"reason": string(reason)}).Inc()
}

func (s *ServerMetrics) ObservePacket(segment commonmetrics.Segment, packetType client.PacketType) {
	commonmetrics.ObservePacket(s.streamPackets, segment, packetType)
}

func (s *ServerMetrics) ObserveStreamErrorNoPacket(segment commonmetrics.Segment, err error) {
	commonmetrics.ObserveStreamErrorNoPacket(s.streamErrors, segment, err)
}

func (s *ServerMetrics) ObserveStreamError(segment commonmetrics.Segment, err error, packetType client.PacketType) {
	commonmetrics.ObserveStreamError(s.streamErrors, segment, err, packetType)
}
