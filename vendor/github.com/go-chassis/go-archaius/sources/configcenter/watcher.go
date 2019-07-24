package configcenter

import (
	"github.com/go-chassis/go-archaius/core"
	"github.com/go-chassis/go-chassis-config"
	"github.com/go-mesh/openlogging"
	"github.com/gorilla/websocket"
	"sync"
)

//DynamicConfigHandler is a struct
type DynamicConfigHandler struct {
	cc             config.Client
	dimensionsInfo string
	EventHandler   *ConfigCenterEventHandler
	dynamicLock    sync.Mutex
	wsDialer       *websocket.Dialer
	wsConnection   *websocket.Conn
}

func newDynConfigHandlerSource(cfgSrc *Handler, callback core.DynamicConfigCallback) (*DynamicConfigHandler, error) {
	eventHandler := newConfigCenterEventHandler(cfgSrc, callback)
	dynCfgHandler := new(DynamicConfigHandler)
	dynCfgHandler.EventHandler = eventHandler
	dynCfgHandler.cc = cfgSrc.cc
	return dynCfgHandler, nil
}
func (dynHandler *DynamicConfigHandler) startDynamicConfigHandler() error {
	err := dynHandler.cc.Watch(
		func(kv map[string]interface{}) {
			dynHandler.EventHandler.OnReceive(kv)
		},
		func(err error) {
			openlogging.Error(err.Error())
		},
	)
	return err

}

//Cleanup cleans particular dynamic configuration Handler up
func (dynHandler *DynamicConfigHandler) Cleanup() error {
	dynHandler.dynamicLock.Lock()
	defer dynHandler.dynamicLock.Unlock()
	if dynHandler.wsConnection != nil {
		dynHandler.wsConnection.Close()
	}
	dynHandler.wsConnection = nil
	return nil
}
