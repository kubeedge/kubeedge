package configcenter

import (
	"github.com/go-chassis/go-archaius/core"
	"github.com/go-mesh/openlogging"
)

//ConfigCenterEventHandler handles a event of a configuration center
type ConfigCenterEventHandler struct {
	ConfigSource *Handler
	Callback     core.DynamicConfigCallback
}

//ConfigCenterEvent stores info about an configuration center event
type ConfigCenterEvent struct {
	Action string `json:"action"`
	Value  string `json:"value"`
}

func newConfigCenterEventHandler(cfgSrc *Handler, callback core.DynamicConfigCallback) *ConfigCenterEventHandler {
	eventHandler := new(ConfigCenterEventHandler)
	eventHandler.ConfigSource = cfgSrc
	eventHandler.Callback = callback
	return eventHandler
}

//OnReceive initializes all necessary components for a configuration center
func (eventHandler *ConfigCenterEventHandler) OnReceive(sourceConfig map[string]interface{}) {

	events, err := eventHandler.ConfigSource.populateEvents(sourceConfig)
	if err != nil {
		openlogging.GetLogger().Error("error in generating event:" + err.Error())
		return
	}

	openlogging.GetLogger().Debugf("event On Receive", events)
	for _, event := range events {
		eventHandler.Callback.OnEvent(event)
	}

	return
}

//OnConnect is a method
func (*ConfigCenterEventHandler) OnConnect() {
	return
}

//OnConnectionClose is a method
func (*ConfigCenterEventHandler) OnConnectionClose() {
	return
}
