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

package inference

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/kubeedge/examples/predictive-maintenance/mapper/pkg/driver"
)

func reading(vib, temp float64) driver.SensorReading {
	return driver.SensorReading{Vibration: vib, Temperature: temp}
}

func TestWarmupPhase(t *testing.T) {
	d := NewDetector()
	for i := 0; i < 4; i++ {
		assert.False(t, d.Analyze(reading(0.5, 45.0)).IsAnomaly)
	}
}

func TestNoAnomalyOnNormalData(t *testing.T) {
	d := NewDetector()
	for i := 0; i < 20; i++ {
		// vary both dims so std is non-trivial
		d.Analyze(reading(0.48+float64(i%5)*0.01, 44.0+float64(i%5)*0.5))
	}
	// 0.50 and 45.0 are the mean — clearly normal
	assert.False(t, d.Analyze(reading(0.50, 45.0)).IsAnomaly)
}

func TestAnomalyDetectedOnSpike(t *testing.T) {
	d := NewDetectorWithParams(20, 2.5)
	for i := 0; i < 20; i++ {
		d.Analyze(reading(0.5, 45.0))
	}
	r := d.Analyze(reading(5.0, 100.0))
	assert.True(t, r.IsAnomaly)
	assert.Greater(t, math.Abs(r.VibrationZScore), 2.5)
}

func TestConfidenceRange(t *testing.T) {
	d := NewDetector()
	for i := 0; i < 25; i++ {
		vib := 0.5
		if i == 20 {
			vib = 99.0
		}
		r := d.Analyze(reading(vib, 45.0))
		assert.GreaterOrEqual(t, r.Confidence, 0.0)
		assert.LessOrEqual(t, r.Confidence, 1.0)
	}
}

func TestWindowStats(t *testing.T) {
	d := NewDetector()
	for i := 0; i < 10; i++ {
		d.Analyze(reading(0.5, 45.0))
	}
	vibMean, _, tempMean, _, n := d.WindowStats()
	require.Equal(t, 10, n)
	assert.InDelta(t, 0.5, vibMean, 0.01)
	assert.InDelta(t, 45.0, tempMean, 0.01)
}

func TestZScore(t *testing.T) {
	assert.InDelta(t, 0.0, zScore(5, 5, 1), 0.001)
	assert.InDelta(t, 1.0, zScore(7, 5, 2), 0.001)
	assert.InDelta(t, -2.0, zScore(1, 5, 2), 0.001)
	assert.InDelta(t, 0.0, zScore(5, 5, 0), 0.001)
}

func TestMeanStd(t *testing.T) {
	w := []driver.SensorReading{{Vibration: 1}, {Vibration: 2}, {Vibration: 3}}
	mean, std := meanStd(w, func(r driver.SensorReading) float64 { return r.Vibration })
	assert.InDelta(t, 2.0, mean, 0.001)
	assert.InDelta(t, 0.8165, std, 0.001)
}

func TestConcurrentAnalyze(t *testing.T) {
	d := NewDetector()
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				d.Analyze(reading(0.5, 45.0))
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
