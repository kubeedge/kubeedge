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

package edgehub

import (
	"io/ioutil"
	"math"

	"github.com/kubeedge/beehive/pkg/core"
)

const inf = math.MaxInt32

type EdgeConfig struct {
	Cloud struct {
		IP          string `yaml:"ip"`
		SignalRange int    `yaml:"signal_range"`
	} `yaml:"cloud"`
	Nodes []struct {
		IP              string `yaml:"ip"`
		DistanceToCloud int    `yaml:"distance_to_cloud"`
	} `yaml:"nodes"`
	Distances []struct {
		Node1    string `yaml:"node1"`
		Node2    string `yaml:"node2"`
		Distance int    `yaml:"distance"`
	} `yaml:"distances"`
}

type LevelMessage struct {
	IP    string `json:"ip"`
	Level int    `json:"level"`
}

func loadEdgeConfig(path string) EdgeConfig {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		klog.Fatalf("Failed to read config file: %v", err)
	}

	var config EdgeConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		//Configuration file parsing error
		klog.Fatalf("Failed to unmarshal config file: %v", err)
	}

	return config
}

func initializeNodeLevels(config EdgeConfig) map[string]int {
	nodeLevels := make(map[string]int)
	nodeLevels[config.Cloud.IP] = 0 // cloud

	// init edge level is infinite
	for _, node := range config.Nodes {
		nodeLevels[node.IP] = inf
	}

	return nodeLevels
}

func broadcastAndUpdateLevels(config EdgeConfig, nodeLevels map[string]int) {
	stable := false
	for !stable {
		stable = true
		// Iterate through each node and update the level
		for _, node := range config.Nodes {
			currentLevel := nodeLevels[node.IP]
			updated := false

			// Check the distance to the cloud node
			if node.DistanceToCloud <= config.Cloud.SignalRange {
				newLevel := nodeLevels[config.Cloud.IP] + 1
				if newLevel < currentLevel {
					nodeLevels[node.IP] = newLevel
					klog.Infof("Sent level message to neighbor: %s, {IP:%s Level:%d}", config.Cloud.IP, node.IP, newLevel)
					updated = true
				}
			}

			// Check the distance to other nodes and update
			for _, dist := range config.Distances {
				if dist.Node1 == node.IP || dist.Node2 == node.IP {
					var neighborIP string
					if dist.Node1 == node.IP {
						neighborIP = dist.Node2
					} else {
						neighborIP = dist.Node1
					}
					neighborLevel := nodeLevels[neighborIP]
					if neighborLevel != inf { // neighbor level is update
						newLevel := neighborLevel + 1
						if newLevel < currentLevel && dist.Distance <= config.Cloud.SignalRange {
							nodeLevels[node.IP] = newLevel
							klog.Infof("Sent level message to neighbor: %s, {IP:%s Level:%d}", neighborIP, node.IP, newLevel)
							updated = true
						}
					}
				}
			}
			if updated {
				stable = false
			}
		}
	}
}

// EdgeHubModule is the Module definition of EdgeHub and implements the Module interface
type EdgeHubModule struct {
	edgeIP     string
	finalLevel int // final level result
}

// Name
func (eh *EdgeHubModule) Name() string {
	return "edgehub"
}

// Group
func (eh *EdgeHubModule) Group() string {
	return "hub"
}

// Enable
func (eh *EdgeHubModule) Enable() bool {
	return true
}

// Start  EdgeHub
func (eh *EdgeHubModule) Start() {
	klog.Infof("Starting EdgeHub for edge: %s", eh.edgeIP)

	// Load configuration file
	config := loadEdgeConfig("config.yaml")

	// initialize level
	nodeLevels := initializeNodeLevels(config)

	// Broadcast and update the level
	broadcastAndUpdateLevels(config, nodeLevels)

	// Final level information
	klog.Infof("Final level information：")
	klog.Infof("Cloud (IP: %s): %dlevel", config.Cloud.IP, nodeLevels[config.Cloud.IP])
	for _, node := range config.Nodes {
		klog.Infof("Node (IP: %s): %dlevel", node.IP, nodeLevels[node.IP])
	}
}

// Register EdgeHub
func RegisterEdgeHub(edgeIP string) {
	core.Register(&EdgeHubModule{edgeIP: edgeIP})
}
