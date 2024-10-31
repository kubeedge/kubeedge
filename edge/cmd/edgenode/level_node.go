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

package main

import (
	"io/ioutil"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
)

type EdgeConfig struct {
	Cloud struct {
		IP          string `yaml:"ip"`
		SignalRange int    `yaml:"signal_range"`
	} `yaml:"cloud"`
	Nodes []struct {
		IP              string `yaml:"ip"`
		DistanceToCloud int    `yaml:"distance_to_cloud"`
	} `yaml:"nodes"`
}

// Read configuration file
func loadEdgeConfig(path string) EdgeConfig {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		klog.Fatalf("Unable to read the configuration file: %v", err)
	}

	var config EdgeConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		klog.Fatalf("Configuration file parsing error: %v", err)
	}

	return config
}

func main() {
	// Load configuration file
	config := loadEdgeConfig("config.yaml")

	// Register the EdgeHub module for each edge node
	for _, node := range config.Nodes {
		edgehub.RegisterEdgeHub(node.IP) // Register each Edge node
	}

	// run Beehive
	core.Run()
}
