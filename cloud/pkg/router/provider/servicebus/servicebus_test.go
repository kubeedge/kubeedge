package servicebus

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/kubeedge/api/apis/rules/v1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/constants"
)

// spyTarget is a test double for provider.Target that records whether
// GoToTarget was invoked.  It never calls into any real infrastructure.
type spyTarget struct {
	called bool
}

func (s *spyTarget) Name() string { return "spy" }
func (s *spyTarget) GoToTarget(_ map[string]interface{}, _ chan struct{}) (interface{}, error) {
	s.called = true
	return nil, fmt.Errorf("spy: GoToTarget should not have been called")
}

func TestServicebusFactoryType(t *testing.T) {
	factory := &servicebusFactory{}
	assert.Equal(t, v1.RuleEndpointTypeServiceBus, factory.Type())
}

func TestServicebusName(t *testing.T) {
	sb := &ServiceBus{}
	assert.Equal(t, constants.ServicebusProvider, sb.Name())
}

func TestServicebusGetSource(t *testing.T) {
	factory := &servicebusFactory{}
	ep := &v1.RuleEndpoint{}

	testCases := []struct {
		name           string
		sourceResource map[string]string
		expectNil      bool
	}{
		{
			name: "valid source",
			sourceResource: map[string]string{
				constants.TargetURL: "http://example.com",
				constants.NodeName:  "test-node",
			},
			expectNil: false,
		},
		{
			name: "missing target_url",
			sourceResource: map[string]string{
				constants.NodeName: "test-node",
			},
			expectNil: true,
		},
		{
			name: "missing node_name",
			sourceResource: map[string]string{
				constants.TargetURL: "http://example.com",
			},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := factory.GetSource(ep, tc.sourceResource)
			if tc.expectNil {
				assert.Nil(t, source)
			} else {
				assert.NotNil(t, source)
			}
		})
	}
}

func TestServicebusGetTarget(t *testing.T) {
	factory := &servicebusFactory{}

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
					Properties: map[string]string{
						"service_port": "8080",
					},
				},
			},
			targetResource: map[string]string{
				constants.Path: "/api/v1",
			},
			expectNil: false,
		},
		{
			name: "missing path",
			ep: &v1.RuleEndpoint{
				Spec: v1.RuleEndpointSpec{
					Properties: map[string]string{},
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

func TestServicebusForward_InvalidMessageType(t *testing.T) {
	sb := &ServiceBus{}
	spy := &spyTarget{}

	_, err := sb.Forward(spy, "not-a-message")

	assert.Error(t, err, "Forward must return an error for a non-*model.Message input")
	assert.False(t, spy.called, "GoToTarget must NOT be called when message type assertion fails")
}

// TestServicebusForward_GetContentDataError is the regression test requested by
// PR #7054 review: when GetContentData() fails, Forward must return an error and
// must never invoke target.GoToTarget().
func TestServicebusForward_GetContentDataError(t *testing.T) {
	sb := &ServiceBus{}
	spy := &spyTarget{}

	// GetContentData returns (data, nil) for string and []byte content;
	// it calls json.Marshal for anything else.  A channel cannot be marshalled
	// so it reliably triggers the error branch.
	msg := model.NewMessage("")
	msg.Content = make(chan struct{})

	_, err := sb.Forward(spy, msg)

	assert.Error(t, err, "Forward must return an error when GetContentData fails")
	assert.False(t, spy.called, "GoToTarget must NOT be called when GetContentData fails")
}
