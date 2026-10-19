package servicebus

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// fakeTarget is a provider.Target that records the data it is given and always
// succeeds, so any error returned by Forward can only come from extracting the
// message content.
type fakeTarget struct {
	received map[string]interface{}
}

func (f *fakeTarget) Name() string { return "fake" }

func (f *fakeTarget) GoToTarget(data map[string]interface{}, _ chan struct{}) (interface{}, error) {
	f.received = data
	return nil, nil
}

func TestForwardContentDataError(t *testing.T) {
	sb := &ServiceBus{}
	msg := model.NewMessage("")
	// A channel value cannot be marshaled to JSON, so GetContentData returns an error.
	msg.Content = make(chan int)

	target := &fakeTarget{}
	resp, err := sb.Forward(target, msg)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Nil(t, target.received, "message should not be forwarded when its content cannot be extracted")
}
