package main

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/klog/v2"
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
