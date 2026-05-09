package filter

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestMessageFilterAddFilterFunc(t *testing.T) {
	messageFilter := &MessageFilter{}

	filterFunc := func(*model.Message) error {
		return nil
	}
	messageFilter.AddFilterFunc(filterFunc)

	assert.Len(t, messageFilter.Filters, 1)
}

func TestMessageFilterProcessFilterWithoutFilters(t *testing.T) {
	messageFilter := &MessageFilter{}
	msg := model.NewMessage("")

	err := messageFilter.ProcessFilter(msg)

	assert.NoError(t, err)
}

func TestMessageFilterProcessFilterInOrder(t *testing.T) {
	messageFilter := &MessageFilter{}
	msg := model.NewMessage("")

	var calls []int
	messageFilter.AddFilterFunc(func(*model.Message) error {
		calls = append(calls, 1)
		return nil
	})
	messageFilter.AddFilterFunc(func(*model.Message) error {
		calls = append(calls, 2)
		return nil
	})

	err := messageFilter.ProcessFilter(msg)

	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2}, calls)
}

func TestMessageFilterProcessFilterStopsOnError(t *testing.T) {
	messageFilter := &MessageFilter{}
	msg := model.NewMessage("")
	expectedErr := errors.New("filter failed")

	calledSecond := false
	messageFilter.AddFilterFunc(func(*model.Message) error {
		return expectedErr
	})
	messageFilter.AddFilterFunc(func(*model.Message) error {
		calledSecond = true
		return nil
	})

	err := messageFilter.ProcessFilter(msg)

	assert.ErrorIs(t, err, expectedErr)
	assert.False(t, calledSecond)
}

func TestMessageFilterProcessFilterReturnsOriginalError(t *testing.T) {
	messageFilter := &MessageFilter{}
	msg := model.NewMessage("")
	expectedErr := errors.New("original error")

	messageFilter.AddFilterFunc(func(*model.Message) error {
		return expectedErr
	})

	err := messageFilter.ProcessFilter(msg)

	assert.Same(t, expectedErr, err)
}
