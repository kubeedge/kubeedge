/*
Copyright 2019 The KubeEdge Authors.

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

package controller

import (
	"encoding/json"
	"strings"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/glog"
	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"

	"github.com/kubeedge/kubeedge/device/bluetooth_mapper/action_manager"
	"github.com/kubeedge/kubeedge/device/bluetooth_mapper/configuration"
	"github.com/kubeedge/kubeedge/device/bluetooth_mapper/data_converter"
	"github.com/kubeedge/kubeedge/device/bluetooth_mapper/helper"
	"github.com/kubeedge/kubeedge/device/bluetooth_mapper/scheduler"
	"github.com/kubeedge/kubeedge/device/bluetooth_mapper/watcher"
)

// constants which can be used to convey topic information
const (
	MapperTopicPrefix              = "$ke/device/bluetooth-mapper/"
	WatcherTopicSuffix             = "/watcher/create"
	SchedulerCreateTopicSuffix     = "/scheduler/create"
	SchedulerDeleteTopicSuffix     = "/scheduler/delete"
	ActionManagerCreateTopicSuffix = "/action-manager/create"
	ActionManagerDeleteTopicSuffix = "/action-manager/delete"
)

var topicMap = make(map[string]MQTT.MessageHandler)

//Config contains the configuration used by the controller
type Config struct {
	Mqtt          configuration.Mqtt          `yaml:"mqtt"`
	Device        configuration.Device        `yaml:"device"`
	Watcher       watcher.Watcher             `yaml:"watcher"`
	Scheduler     scheduler.Scheduler         `yaml:"scheduler"`
	ActionManager actionmanager.ActionManager `yaml:"action-manager"`
	Converter     dataconverter.Converter     `yaml:"data-converter"`
}

// initTopicMap initializes topics to their respective handler functions
func (c *Config) initTopicMap() {
	topicMap[MapperTopicPrefix+c.Device.ID+WatcherTopicSuffix] = c.handleWatchMessage
	topicMap[MapperTopicPrefix+c.Device.ID+SchedulerCreateTopicSuffix] = c.handleScheduleCreateMessage
	topicMap[MapperTopicPrefix+c.Device.ID+SchedulerDeleteTopicSuffix] = c.handleScheduleDeleteMessage
	topicMap[MapperTopicPrefix+c.Device.ID+ActionManagerCreateTopicSuffix] = c.handleActionCreateMessage
	topicMap[MapperTopicPrefix+c.Device.ID+ActionManagerDeleteTopicSuffix] = c.handleActionDeleteMessage
}

//Start starts the controller of the mapper
func (c *Config) Start() {
	c.initTopicMap()
	helper.MqttConnect(c.Mqtt.Mode, c.Mqtt.InternalServer, c.Mqtt.Server)
	subscribeAllTopics()
	helper.ControllerWg.Add(1)
	device, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		glog.Fatalf("Failed to open device, err: %s\n", err)
		return
	}
	go c.Watcher.Initiate(device, c.Device.Name, c.Device.ID, c.ActionManager.Actions, c.Converter)

	<-watcher.DeviceConnected
	for _, action := range c.ActionManager.Actions {
		if action.PerformImmediately {
			action.PerformOperation(c.Converter.DataRead)
		}
	}

	for _, schedule := range c.Scheduler.Schedules {
		helper.ControllerWg.Add(1)
		go schedule.ExecuteSchedule(c.ActionManager.Actions, c.Converter.DataRead, c.Device.ID)
	}
	helper.ControllerWg.Wait()
}

//subscribeAllTopics subscribes to mqtt topics associated with mapper
func subscribeAllTopics() {
	for key, value := range topicMap {
		helper.TokenClient = helper.Client.Subscribe(key, 0, value)
		if helper.TokenClient.Wait() && helper.TokenClient.Error() != nil {
			glog.Errorf("subscribe() Error in topic: %s is: %s", key, helper.TokenClient.Error())
		}
	}
}

//handleWatchMessage is the MQTT handler function for changing watcher configuration at runtime
func (c *Config) handleWatchMessage(client MQTT.Client, message MQTT.Message) {
	newWatch := watcher.Watcher{}
	err := json.Unmarshal(message.Payload(), &newWatch)
	if err != nil {
		glog.Errorf("Error in unmarshalling:  %s", err)
	}
	c.Watcher = newWatch
	configuration.Config.Watcher = c.Watcher
	glog.Infof("New watcher has been started")
	glog.Infof("New Watcher: %v", c.Watcher)
}

//handleScheduleCreateMessage is the MQTT handler function for adding schedules at runtime
func (c *Config) handleScheduleCreateMessage(client MQTT.Client, message MQTT.Message) {
	newSchedules := []scheduler.Schedule{}
	err := json.Unmarshal(message.Payload(), &newSchedules)
	if err != nil {
		glog.Errorf("Error in unmarshalling: %s", err)
	}
	for _, newSchedule := range newSchedules {
		scheduleExists := false
		for scheduleIndex, schedule := range c.Scheduler.Schedules {
			if schedule.Name == newSchedule.Name {
				c.Scheduler.Schedules[scheduleIndex] = newSchedule
				scheduleExists = true
				break
			}
		}
		if scheduleExists {
			c.Scheduler.Schedules = append(c.Scheduler.Schedules, newSchedule)
			glog.Infof("Schedule: %s has been updated", newSchedule.Name)
			glog.Infof("Updated Schedule: %v", newSchedule)
		} else {
			glog.Infof("Schedule: %s has been added", newSchedule.Name)
			glog.Infof("New Schedule: %v", newSchedule)
		}
		configuration.Config.Scheduler = c.Scheduler
		newSchedule.ExecuteSchedule(c.ActionManager.Actions, c.Converter.DataRead, c.Device.ID)
	}
}

//handleScheduleDeleteMessage is the MQTT handler function for deleting schedules at runtime
func (c *Config) handleScheduleDeleteMessage(client MQTT.Client, message MQTT.Message) {
	schedulesToBeDeleted := []scheduler.Schedule{}
	err := json.Unmarshal(message.Payload(), &schedulesToBeDeleted)
	if err != nil {
		glog.Errorf("Error in unmarshalling:  %s", err)
	}
	for _, scheduleToBeDeleted := range schedulesToBeDeleted {
		scheduleExists := false
		for index, schedule := range c.Scheduler.Schedules {
			if strings.ToUpper(schedule.Name) == strings.ToUpper(scheduleToBeDeleted.Name) {
				scheduleExists = true
				copy(c.Scheduler.Schedules[index:], c.Scheduler.Schedules[index+1:])
				c.Scheduler.Schedules = c.Scheduler.Schedules[:len(c.Scheduler.Schedules)-1]
				break
			}
		}
		configuration.Config.Scheduler = c.Scheduler
		if !scheduleExists {
			glog.Errorf("Schedule: %s does not exist", scheduleToBeDeleted.Name)
		} else {
			glog.Infof("Schedule: %s has been deleted ", scheduleToBeDeleted.Name)
		}
	}
}

//handleActionCreateMessage MQTT handler function for adding actions at runtime
func (c *Config) handleActionCreateMessage(client MQTT.Client, message MQTT.Message) {
	newActions := []actionmanager.Action{}
	err := json.Unmarshal(message.Payload(), &newActions)
	if err != nil {
		glog.Errorf("Error in unmarshalling:  %s", err)
	}
	for _, newAction := range newActions {
		actionExists := false
		for actionIndex, action := range c.ActionManager.Actions {
			if action.Name == newAction.Name {
				c.ActionManager.Actions[actionIndex] = newAction
				actionExists = true
				break
			}
		}
		if actionExists {
			c.ActionManager.Actions = append(c.ActionManager.Actions, newAction)
			glog.Infof("Action: %s has been updated", newAction.Name)
			glog.Infof("Updated Action: %v", newAction)
		} else {
			glog.Infof("Action: %s has been added ", newAction.Name)
			glog.Infof("New Action: %v", newAction)
		}
		configuration.Config.ActionManager = c.ActionManager
		if newAction.PerformImmediately {
			newAction.PerformOperation(c.Converter.DataRead)
		}
	}
}

//handleActionDeleteMessage MQTT handler function for deleting actions at runtime
func (c *Config) handleActionDeleteMessage(client MQTT.Client, message MQTT.Message) {
	actionsToBeDeleted := []actionmanager.Action{}
	err := json.Unmarshal(message.Payload(), &actionsToBeDeleted)
	if err != nil {
		glog.Errorf("Error in unmarshalling:  %s", err)
	}
	for _, actionToBeDeleted := range actionsToBeDeleted {
		actionExists := false
		for index, action := range c.ActionManager.Actions {
			if strings.ToUpper(action.Name) == strings.ToUpper(actionToBeDeleted.Name) {
				actionExists = true
				copy(c.ActionManager.Actions[index:], c.ActionManager.Actions[index+1:])
				c.ActionManager.Actions = c.ActionManager.Actions[:len(c.ActionManager.Actions)-1]
				break
			}
		}
		configuration.Config.ActionManager = c.ActionManager
		if !actionExists {
			glog.Errorf("Action: %s did not exist", actionToBeDeleted.Name)
		} else {
			glog.Infof("Action: %s has been deleted ", actionToBeDeleted.Name)
		}
	}
}
