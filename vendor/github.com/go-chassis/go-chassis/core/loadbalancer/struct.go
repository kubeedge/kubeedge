package loadbalancer

import (
	"time"
)

// ProtocolStats store protocol stats
type ProtocolStats struct {
	Latency    []time.Duration
	Addr       string
	AvgLatency time.Duration
}

// CalculateAverageLatency make avg latency
func (ps *ProtocolStats) CalculateAverageLatency() {
	var sum time.Duration
	for i := 0; i < len(ps.Latency); i++ {
		sum = sum + ps.Latency[i]
	}
	if len(ps.Latency) == 0 {
		return
	}
	ps.AvgLatency = time.Duration(sum.Nanoseconds() / int64(len(ps.Latency)))
}

// SaveLatency save latest 10 record
func (ps *ProtocolStats) SaveLatency(l time.Duration) {
	if len(ps.Latency) >= 10 {
		//save latest 10 latencies
		ps.Latency = ps.Latency[1:]
	}
	ps.Latency = append(ps.Latency, l)
}
