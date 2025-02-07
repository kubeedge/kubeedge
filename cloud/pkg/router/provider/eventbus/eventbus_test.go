package eventbus

import (
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/kubeedge/api/apis/rules/v1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/constants"
)

func TestPathJoin(t *testing.T) {
	var s1, s2 string
	s1 = fmt.Sprintf("%s/node/%s/%s/%s", "bus", "nodeName", "namespace", "subTopic")
	s2 = path.Join("bus/node", "nodeName", "namespace", "subTopic")
	if s1 != s2 {
		t.Fatalf("expected: %s, actual: %s", s1, s2)
	}

	s1 = fmt.Sprintf("node/%s/%s/%s", "nodeName", "namespace", "subTopic")
	s2 = path.Join("node", "nodeName", "namespace", "subTopic")
	if s1 != s2 {
		t.Fatalf("expected: %s, actual: %s", s1, s2)
	}
}

func TestEventbusFactoryType(t *testing.T) {
	factory := &eventbusFactory{}
	assert.Equal(t, v1.RuleEndpointTypeEventBus, factory.Type())
}

func TestGetSource(t *testing.T) {
	factory := &eventbusFactory{}
	testCases := []struct {
		name           string
		ep             *v1.RuleEndpoint
		sourceResource map[string]string
		expectNil      bool
	}{
		{
			name: "valid source",
			ep: &v1.RuleEndpoint{
				Spec: v1.RuleEndpointSpec{
					RuleEndpointType: v1.RuleEndpointTypeEventBus,
				},
			},
			sourceResource: map[string]string{
				constants.Topic:    "test-topic",
				constants.NodeName: "test-node",
			},
			expectNil: false,
		},
		{
			name: "missing topic",
			ep: &v1.RuleEndpoint{
				Spec: v1.RuleEndpointSpec{
					RuleEndpointType: v1.RuleEndpointTypeEventBus,
				},
			},
			sourceResource: map[string]string{
				constants.NodeName: "test-node",
			},
			expectNil: true,
		},
		{
			name: "missing node name",
			ep: &v1.RuleEndpoint{
				Spec: v1.RuleEndpointSpec{
					RuleEndpointType: v1.RuleEndpointTypeEventBus,
				},
			},
			sourceResource: map[string]string{
				constants.Topic: "test-topic",
			},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := factory.GetSource(tc.ep, tc.sourceResource)
			if tc.expectNil {
				assert.Nil(t, source)
			} else {
				assert.NotNil(t, source)
			}
		})
	}
}

func TestGetTarget(t *testing.T) {
	factory := &eventbusFactory{}
	testCases := []struct {
		name           string
		ep             *v1.RuleEndpoint
		targetResource map[string]string
		expectNil      bool
	}{
		{
			name: "valid target",
			ep: &v1.RuleEndpoint{
				Spec: v1.RuleEndpointSpec{
					RuleEndpointType: v1.RuleEndpointTypeEventBus,
				},
			},
			targetResource: map[string]string{
				"topic": "test-topic",
			},
			expectNil: false,
		},
		{
			name: "missing topic",
			ep: &v1.RuleEndpoint{
				Spec: v1.RuleEndpointSpec{
					RuleEndpointType: v1.RuleEndpointTypeEventBus,
				},
			},
			targetResource: map[string]string{},
			expectNil:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target := factory.GetTarget(tc.ep, tc.targetResource)
			if tc.expectNil {
				assert.Nil(t, target)
			} else {
				assert.NotNil(t, target)
			}
		})
	}
}

func TestName(t *testing.T) {
	eb := &EventBus{}
	assert.Equal(t, constants.EventbusProvider, eb.Name())
}

func TestForward(t *testing.T) {
	eb := &EventBus{}
	testCases := []struct {
		name        string
		data        interface{}
		targetData  map[string]interface{}
		expectError bool
	}{
		{
			name:        "invalid message type",
			data:        "invalid",
			expectError: true,
		},
		{
			name: "valid message type with invalid content",
			data: func() interface{} {
				msg := model.NewMessage("")
				msg.Content = "test-content"
				return msg
			}(),
			expectError: true,
		},
		{
			name: "valid message with valid GoToTarget data",
			data: func() interface{} {
				msg := model.NewMessage("")
				msg.Content = map[string]interface{}{
					"messageID": "test-id",
					"nodeName":  "test-node",
					"data":      []byte("test-data"),
				}
				return msg
			}(),
			expectError: true, // Will error due to missing session manager
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target := &EventBus{
				pubTopic: "test-topic",
			}
			_, err := eb.Forward(target, tc.data)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGoToTarget(t *testing.T) {
	eb := &EventBus{
		pubTopic: "test-topic",
	}
	testCases := []struct {
		name        string
		data        map[string]interface{}
		expectError bool
	}{
		{
			name: "valid data with param",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  "test-node",
				"data":      []byte("test-data"),
				"param":     "test-param",
			},
			expectError: true, // Will error due to missing session manager
		},
		{
			name: "valid data without param",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  "test-node",
				"data":      []byte("test-data"),
			},
			expectError: true, // Will error due to missing session manager
		},
		{
			name: "missing messageID",
			data: map[string]interface{}{
				"nodeName": "test-node",
				"data":     []byte("test-data"),
			},
			expectError: true,
		},
		{
			name: "missing nodeName",
			data: map[string]interface{}{
				"messageID": "test-id",
				"data":      []byte("test-data"),
			},
			expectError: true,
		},
		{
			name: "missing data",
			data: map[string]interface{}{
				"messageID": "test-id",
				"nodeName":  "test-node",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := eb.GoToTarget(tc.data, nil)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildAndLogError(t *testing.T) {
	err := buildAndLogError("test-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test-key")
}
