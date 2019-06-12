/*
 * Copyright 2017 Huawei Technologies Co., Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
* Created by on 2017/7/19.
 */
// Package main for examples
package hack

import (
	"fmt"
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-archaius/core"
	"github.com/go-chassis/go-archaius/sources/file-source"
	"github.com/go-mesh/openlogging"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//EventListener
type EventListener struct {
	Name string
}

//ConfigFactory
var ConfigFactory archaius.ConfigurationFactory

func main() {
	// init logger for archaius
	//logger = log.Init()
	//logger.Formatter = new(log.JSONFormatter)
	//logger.Level = log.InfoLevel

	// create go-archaius object
	configFactory, err := archaius.NewConfigFactory()
	if err != nil {
		openlogging.GetLogger().Error("Error:" + err.Error())
	}
	ConfigFactory = configFactory
	// init go-archaius
	err = ConfigFactory.Init()
	if err != nil {
		openlogging.GetLogger().Error("Error:" + err.Error())
	}

	// create event receiver for configuration changes.
	// event receiver must implement core.EventListener interface
	eventListener := EventListener{Name: "eventListener1"}
	// register event receiver to go-archaius
	// regular expression support in key
	ConfigFactory.RegisterListener(eventListener, "s*")

	// get default configurations from go archaius
	// default configurations involve 1. commandline arguments 2. environment variables
	config := ConfigFactory.GetConfigurations()
	openlogging.GetLogger().Infof("======================== Default config====================== ",
		config,
		"===========================================================\n")

	// create file source object
	fSource := filesource.NewFileSource()
	// add file in file source.
	// file can be regular yaml file or directory like fSource.AddFile("./conf", 0)
	// second argument is priority of file
	fSource.AddFile("./conf/name.yaml", 0, nil)
	// add file source to go-archaius
	ConfigFactory.AddSource(fSource)

	// get default and file source configurations
	config = configFactory.GetConfigurations()
	openlogging.GetLogger().Infof("======================== Default and File source config====================== ",
		config,
		"===========================================================\n")

	// // Steps to add config center source
	//configCenters := make([]string, 0)
	//configCenters = append(configCenters, "9.93.0.221:30103")
	//memDiscovery := memberdiscovery.NewMemDiscovery(logger.WithFields(log.Fields{
	//	"source": "member-dis",
	//}))
	//memDiscovery.Init(configCenters)
	//dimensionsInfo := "service1"
	//logger.Debug(`config client init with ` + dimensionsInfo + ` dimension info`)
	//csloger := loger.NewLogger(logger.WithFields(log.Fields{
	//	"source": "config-sebeter-logge",
	//}))
	//configCenterSource, err := configcentersource.NewConfigCenterSource(memDiscovery, dimensionsInfo, csloger)
	//if err != nil {
	//	logger.Error("invalid server uri format. ignoring config center client initialization")
	//	return
	//}
	//ConfigFactory.AddSource(configCenterSource)
	//config = ConfigFactory.GetConfigurations()
	//logger.Info("======================== Get all configurations ====================== ",
	//				config,
	//	    "===========================================================\n")

	time.Sleep(2 * time.Second)

	err = ConfigFactory.DeInit()
	if err != nil {
		openlogging.GetLogger().Error("Error:" + err.Error())
	}

	config = ConfigFactory.GetConfigurations()
	openlogging.GetLogger().Infof("======================== After Deinit config======================\n ",
		config,
		"\n========================\n")

	//second argument is priority of file
	fSource.AddFile("./conf/name.yaml", 0, nil)
	// add file source to go-`archaius
	ConfigFactory.AddSource(fSource)

	time.Sleep(1 * time.Second)
	config = ConfigFactory.GetConfigurations()
	openlogging.GetLogger().Infof("\n \n======================== after adding file source: ======================== \n", config, "======================== \n ")

	//// adding config center source
	//ConfigFactory.AddSource(configCenterSource)
	//config = ConfigFactory.GetConfigurations()
	//logger.Infoln("\n \n ======================== after adding config center source : ======================== \n", config, "======================== \n")

	// can check for key existence
	key := "name"
	if ConfigFactory.IsKeyExist(key) {
		openlogging.GetLogger().Infof(key, " key exist")
	}

	name, err := ConfigFactory.GetValue(key).ToString()
	if err != nil {
		openlogging.GetLogger().Error("get value failed with error " + err.Error())
	} else {
		openlogging.GetLogger().Infof("Reterived value of name is ", name)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	fmt.Println("exit graceful")

}

//Event is a method get config value and logs it
func (e EventListener) Event(event *core.Event) {

	configValue := ConfigFactory.GetConfigurationByKey(event.Key)
	openlogging.GetLogger().Infof("config value ", event.Key, " | ", configValue)
}
