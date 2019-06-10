package eventsystem_test

import (
	"github.com/go-chassis/go-archaius/core"
	"github.com/go-chassis/go-archaius/core/config-manager"
	"github.com/go-chassis/go-archaius/core/event-system"
	"github.com/go-chassis/go-archaius/sources/test-source"
	"testing"
	"time"
)

type EListener struct {
	Name      string
	EventName string
}

func (e *EListener) Event(event *core.Event) {
	e.EventName = event.Key
}

func TestEventLoop1(t *testing.T) {
	// test event
	event := "ddd"
	testConfig := map[string]interface{}{"aaa": "111", "bbb": "222"}
	testSource := testsource.NewTestSource(testConfig)

	dispatcher := eventsystem.NewDispatcher()
	cm := configmanager.NewConfigurationManager(dispatcher)

	t.Log("Test eventsystem.go")

	cm.AddSource(testSource, configmanager.DefaultPriority)

	time.Sleep(1 * time.Second)
	t.Log(" Register Listener")
	eventListener := &EListener{Name: "eventListener"}
	dispatcher.RegisterListener(eventListener, event)

	t.Log("create event")
	testsource.AddConfig(event, "value1")
	t.Log("verifying created event")
	if len(testsource.GetTestSource().Configuration) == 2 {
		t.Error("Config items error before refresh after update source")
	}

	time.Sleep(1 * time.Second)

	if eventListener.EventName != event {
		t.Error("Error while generating event")
	}

	dispatcher.UnRegisterListener(eventListener, event)
	t.Log("UnRegister Listener")
	testsource.CleanupTestSource()
}

func TestDispatchEvent(t *testing.T) {

	//dispatcher

	dispatcher := eventsystem.NewDispatcher()
	var event *core.Event = nil
	err := dispatcher.DispatchEvent(event)
	if err == nil {
		t.Error("Dispatcher failed to identify the nil event")
	}

	eventListener1 := &EListener{Name: "eventListener"}
	eventListener2 := &EListener{Name: "eventListener"}
	eventListener3 := &EListener{Name: "eventListener"}
	err = dispatcher.RegisterListener(eventListener1, "*")

	event = &core.Event{Key: "TestKey", Value: "TestValue"}
	err = dispatcher.DispatchEvent(event)
	if err != nil {
		t.Error("dispatches the event for regular expresssion failed key")
	}

	dispatcher.RegisterListener(eventListener2, "Key1")
	dispatcher.RegisterListener(eventListener3, "Key1")

	//unregister

	var listener core.EventListener = nil
	//supplying nil listener
	err = dispatcher.UnRegisterListener(listener, "key")
	if err == nil {
		t.Error("event system processing on nil listener")
	}

	err = dispatcher.UnRegisterListener(eventListener1, "unregisteredkey")
	if err != nil {
		t.Error("event system unable to identify the unregisteredkey")
	}

	err = dispatcher.UnRegisterListener(eventListener2, "Key1")
	if err != nil {
		t.Error("event system unable to identify the unregisteredkey")
	}

	dispatcher.UnRegisterListener(eventListener3, "Key1")
	dispatcher.UnRegisterListener(eventListener1, "*")

	//register

	t.Log("supplying nil listener")
	err = dispatcher.RegisterListener(listener, "key")
	if err == nil {
		t.Error("Event system working on nil listener")
	}

	err = dispatcher.RegisterListener(eventListener3, "Key1")
	if err != nil {
		t.Error("Event system working on nil listener")
	}

	t.Log("duplicate registration")
	err = dispatcher.RegisterListener(eventListener3, "Key1")
	if err != nil {
		t.Error("Failed to detect the duplicate registration")
	}

	dispatcher.UnRegisterListener(eventListener3, "Key1")

}
