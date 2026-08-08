/*
Copyright 2024 The KubeEdge Authors.

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

package driver

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/examples/predictive-maintenance/mapper/pkg/config"
)

// SensorReading is one sensor snapshot.
type SensorReading struct {
	Vibration   float64 `json:"vibration"`
	Temperature float64 `json:"temperature"`
	Timestamp   int64   `json:"timestamp"`
	IsAnomaly   bool    `json:"isAnomaly"`
}

// VirtualSensor simulates a factory sensor.
type VirtualSensor struct {
	cfg    config.SensorConfig
	rng    *rand.Rand
	mu     sync.Mutex
	window []SensorReading
}

// NewVirtualSensor returns a new virtual sensor.
func NewVirtualSensor(cfg config.SensorConfig) *VirtualSensor {
	return &VirtualSensor{
		cfg:    cfg,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
		window: make([]SensorReading, 0, 20),
	}
}

// Read generates one sensor reading.
func (v *VirtualSensor) Read() SensorReading {
	v.mu.Lock()
	defer v.mu.Unlock()

	isAnomaly := v.rng.Float64() < v.cfg.AnomalyProbability

	// Box-Muller Gaussian noise
	u1 := v.rng.Float64()
	u2 := v.rng.Float64()
	noise := math.Sqrt(-2*math.Log(u1+1e-10)) * math.Cos(2*math.Pi*u2)

	vibration := v.cfg.BaseVibration + noise*v.cfg.NoiseLevel
	temperature := v.cfg.BaseTemperature + noise*v.cfg.NoiseLevel*10

	if isAnomaly {
		vibration *= v.cfg.AnomalyMultiplier
		temperature += 20.0 * v.cfg.AnomalyMultiplier / 3.5
		klog.V(2).Infof("anomaly injected: vib=%.4f temp=%.2f", vibration, temperature)
	}

	reading := SensorReading{
		Vibration:   math.Round(math.Max(0, vibration)*10000) / 10000,
		Temperature: math.Round(math.Max(0, temperature)*100) / 100,
		Timestamp:   time.Now().UnixMilli(),
		IsAnomaly:   isAnomaly,
	}

	if len(v.window) >= 20 {
		v.window = v.window[1:]
	}
	v.window = append(v.window, reading)
	return reading
}

// RecentReadings returns the rolling window copy.
func (v *VirtualSensor) RecentReadings() []SensorReading {
	v.mu.Lock()
	defer v.mu.Unlock()
	result := make([]SensorReading, len(v.window))
	copy(result, v.window)
	return result
}
