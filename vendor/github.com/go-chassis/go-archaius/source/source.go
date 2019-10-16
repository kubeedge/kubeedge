package source

import "github.com/go-chassis/go-archaius/event"

// ConfigSource get key values from a system, like file system, key value store, memory etc
type ConfigSource interface {
	Set(key string, value interface{}) error
	Delete(key string) error
	GetConfigurations() (map[string]interface{}, error)
	GetConfigurationByKey(string) (interface{}, error)
	Watch(EventHandler) error
	GetPriority() int
	SetPriority(priority int)
	Cleanup() error
	GetSourceName() string

	AddDimensionInfo(labels map[string]string) error
}

// EventHandler handles config change event
type EventHandler interface {
	OnEvent(event *event.Event)
}
