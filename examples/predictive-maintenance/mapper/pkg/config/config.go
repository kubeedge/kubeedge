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

package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds mapper settings.
type Config struct {
	DMIConfig         DMIConfig    `json:"dmi"`
	SensorConfig      SensorConfig `json:"sensor"`
	ReportIntervalSec int          `json:"reportIntervalSec"`
}

// DMIConfig holds EdgeCore connection info.
type DMIConfig struct {
	SocketPath      string `json:"socketPath"`
	MapperName      string `json:"mapperName"`
	Protocol        string `json:"protocol"`
	DeviceNamespace string `json:"deviceNamespace"`
	DeviceName      string `json:"deviceName"`
}

// SensorConfig holds simulation parameters.
type SensorConfig struct {
	BaseVibration      float64 `json:"baseVibration"`
	BaseTemperature    float64 `json:"baseTemperature"`
	NoiseLevel         float64 `json:"noiseLevel"`
	AnomalyProbability float64 `json:"anomalyProbability"`
	AnomalyMultiplier  float64 `json:"anomalyMultiplier"`
}

// DefaultConfig returns factory floor defaults.
func DefaultConfig() *Config {
	return &Config{
		DMIConfig: DMIConfig{
			SocketPath:      "/var/lib/kubeedge/dmi.sock",
			MapperName:      "predictive-maintenance-mapper",
			Protocol:        "virtual-sensor",
			DeviceNamespace: "default",
			DeviceName:      "factory-sensor-01",
		},
		SensorConfig: SensorConfig{
			BaseVibration:      0.5,
			BaseTemperature:    45.0,
			NoiseLevel:         0.05,
			AnomalyProbability: 0.05,
			AnomalyMultiplier:  3.5,
		},
		ReportIntervalSec: 5,
	}
}

// Load reads config from CONFIG_PATH or uses defaults.
func Load() (*Config, error) {
	cfg := DefaultConfig()
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
