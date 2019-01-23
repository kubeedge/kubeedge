package eventbus

import (
	"testing"

	MQTT "github.com/eclipse/paho.mqtt.golang"

	"github.com/kubeedge/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/common/util"
	mqttBus "github.com/kubeedge/kubeedge/pkg/eventbus/mqtt"
)

// coreContext is beehive context used for communication between modules.
var coreContext *context.Context

// Init_Test function to create mqtt client
func Init_Test() {
	opts := MQTT.NewClientOptions().AddBroker("tcp://127.0.0.1:1884").SetClientID("test").SetCleanSession(true)
	Pubcli := MQTT.NewClient(opts)
	Subcli := MQTT.NewClient(opts)
	hub := &mqttBus.MQTTClient{
		MQTTUrl: "tcp://127.0.0.1:1884",
		PubCli:  Pubcli,
		SubCli:  Subcli,
	}
	mqttBus.MQTTHub = hub
	hub.InitSubClient()
	hub.InitPubClient()
}

// TestName is function to test Name().
func TestName(t *testing.T) {
	coreContext = context.GetContext("channel")
	tests := []struct {
		name     string
		context  *context.Context
		mqttMode int
		want     string
	}{
		{
			name:    "test-name",
			context: coreContext,
			want:    "eventbus",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &eventbus{
				context:  tt.context,
				mqttMode: tt.mqttMode,
			}
			if got := e.Name(); got != tt.want {
				t.Errorf("eventbus.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGroup is function to test Group().
func TestGroup(t *testing.T) {
	coreContext = context.GetContext("channel")
	tests := []struct {
		name     string
		context  *context.Context
		mqttMode int
		want     string
	}{
		{
			name:    "test-group",
			context: coreContext,
			want:    "bus",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &eventbus{
				context:  tt.context,
				mqttMode: tt.mqttMode,
			}
			if got := e.Group(); got != tt.want {
				t.Errorf("eventbus.Group() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStart is function to test Start().
func TestStart(t *testing.T) {
	coreContext := context.GetContext("channel")
	util.LoadConfig()
	core.Register(&eventbus{})
	event := eventbus{}
	coreContext.AddModule(event.Name())
	coreContext.AddModuleGroup(event.Name(), event.Group())
	coreContext = context.GetContext("channel")
	tests := []struct {
		name     string
		context  *context.Context
		mqttMode int
	}{
		{
			name:     "eventbus start",
			context:  coreContext,
			mqttMode: 1,
		},
		{
			name:     "eventbus start",
			context:  coreContext,
			mqttMode: 2,
		},
		{
			name:     "eventbus start",
			context:  coreContext,
			mqttMode: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &eventbus{
				context:  tt.context,
				mqttMode: tt.mqttMode,
			}
			var test model.Message
			test.Content = "subscribe"
			test.Router.Operation = "message"
			go coreContext.Send(event.Name(), test)
			go eb.Start(tt.context)

		})
	}
}

// TestCleanup is function to test cleanup
func TestCleanup(t *testing.T) {
	coreContext = context.GetContext("channel")
	tests := []struct {
		name     string
		context  *context.Context
		mqttMode int
	}{
		{
			name:     "eventbus cleanup",
			context:  coreContext,
			mqttMode: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &eventbus{
				context:  tt.context,
				mqttMode: tt.mqttMode,
			}
			eb.Cleanup()
		})
	}
}

// Test_pubMQTT is a function to pub msg to mqtt broker.
func Test_pubMQTT(t *testing.T) {
	Init_Test()
	tests := []struct {
		name    string
		topic   string
		payload []byte
	}{
		{
			name:    "eventbus-pubMQ",
			topic:   "$hw/events/upload/#",
			payload: []byte("abcdef"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pubMQTT(tt.topic, tt.payload)
		})
	}
}

//Test Function to publish cloud Message to Edge
func Test_eventbus_pubCloudMsgToEdge(t *testing.T) {
	Init_Test()
	coreContext := context.GetContext("channel")
	util.LoadConfig()
	core.Register(&eventbus{})
	event := eventbus{}
	coreContext.AddModule(event.Name())
	coreContext.AddModuleGroup(event.Name(), event.Group())
	coreContext = context.GetContext("channel")
	tests := []struct {
		name     string
		context  *context.Context
		mqttMode int
		topic    string
	}{
		{
			name:     "eventbus-publishing cloud message to edge",
			context:  coreContext,
			mqttMode: 1,
			topic:    "$hw/events/upload/#",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &eventbus{
				context:  tt.context,
				mqttMode: tt.mqttMode,
			}
			var test model.Message
			test.Content = "subscribe operation"
			test.Router.Operation = "subscribe"
			go coreContext.Send(event.Name(), test)
			go eb.pubCloudMsgToEdge()

		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &eventbus{
				context:  tt.context,
				mqttMode: tt.mqttMode,
			}
			var test model.Message
			test.Content = "message operation"
			test.Router.Operation = "message"
			go coreContext.Send(event.Name(), test)
			go eb.pubCloudMsgToEdge()

		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &eventbus{
				context:  tt.context,
				mqttMode: tt.mqttMode,
			}
			var test model.Message
			test.Content = "get-result operation"
			test.Router.Operation = "get_result"
			go coreContext.Send(event.Name(), test)
			go eb.pubCloudMsgToEdge()

		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &eventbus{
				context:  tt.context,
				mqttMode: tt.mqttMode,
			}
			//Message Struct
			var test model.Message
			test.Content = "publish operation"
			test.Router.Operation = "publish"

			go coreContext.Send(event.Name(), test)
			go eb.pubCloudMsgToEdge()

		})
	}
}

//Publish message to the broker
func Test_eventbus_publish(t *testing.T) {
	Init_Test()
	coreContext = context.GetContext("channel")

	tests := []struct {
		name     string
		context  *context.Context
		mqttMode int
		topic    string
		payload  []byte
	}{
		{
			name:     "eventbus-publish",
			context:  coreContext,
			mqttMode: 1,
			topic:    "$hw/events/device/+/state/update",
			payload:  []byte("abcdef"),
		},
		//payload is empty
		{
			name:     "eventbus-publish",
			context:  coreContext,
			mqttMode: 1,
			topic:    "$hw/events/upload/#",
			payload:  []byte(""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &eventbus{
				context:  tt.context,
				mqttMode: tt.mqttMode,
			}
			go eb.publish(tt.topic, tt.payload)
		})
	}
}

//subscribe topic to the broker
func Test_eventbus_subscribe(t *testing.T) {
	coreContext = context.GetContext("channel")
	tests := []struct {
		name     string
		context  *context.Context
		mqttMode int
		topic    string
	}{
		{
			name:     "eventbus-subscribe",
			context:  coreContext,
			mqttMode: 1,
			topic:    "$hw/events/device/+/state/update",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &eventbus{
				context:  tt.context,
				mqttMode: tt.mqttMode,
			}
			go eb.subscribe(tt.topic)
		})
	}
}
