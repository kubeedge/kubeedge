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

package cloudhub

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
)

type CloudConfig struct {
	Cloud struct {
		IP          string `yaml:"ip"`
		SignalRange int    `yaml:"signal_range"`
	} `yaml:"cloud"`
}

type LevelMessage struct {
	IP    string `json:"ip"`
	Level int    `json:"level"`
}

// read config
func loadCloudConfig() CloudConfig {
	data, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		klog.Fatalf("Failed to read config file: %v", err)
	}

	var config CloudConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		klog.Fatalf("Failed to unmarshal config file: %v", err)
	}

	return config
}

type CloudHubModule struct{}

// Name
func (ch *CloudHubModule) Name() string {
	return "cloudhub"
}

// Group
func (ch *CloudHubModule) Group() string {
	return "hub"
}

// Enable
func (ch *CloudHubModule) Enable() bool {
	return true
}

// Start  CloudHub
func (ch *CloudHubModule) Start() {
	klog.Infof("Starting CloudHub...")

	// load config
	config := loadCloudConfig()

	// cloud IP and init level
	cloudIP := config.Cloud.IP
	cloudLevel := 0

	// Periodically broadcast layer information about cloud nodes
	go func() {
		for {
			time.Sleep(10 * time.Second)

			// Build level message
			msg := LevelMessage{
				IP:    cloudIP,
				Level: cloudLevel,
			}

			// Sends a broadcast to all edge nodes
			broadcastLevelMessage(msg)
		}
	}()
}

// Sends hierarchical messages to edge nodes
func broadcastLevelMessage(msg LevelMessage) {
	messageBody, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("Failed to marshal level message: %v", err)
		return
	}

	// Use BuildBody to set the message content
	cloudHubMessage := model.NewMessage("").FillBody(messageBody)

	// Sending messages via EdgeController
	beehiveContext.Send("edgehub", *cloudHubMessage)
	klog.Infof("Broadcasted level message to edges: %+v", msg)
}

// Register the CloudHub module
func RegisterCloudHub() {
	core.Register(&CloudHubModule{})
}
