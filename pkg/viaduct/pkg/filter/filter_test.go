package filter

import (
    "errors"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/kubeedge/beehive/pkg/core/model"
)

func TestMessageFilter(t *testing.T) {
    t.Run("Empty_Filter", func(t *testing.T) {
        filter := &MessageFilter{}
        msg := &model.Message{}
        
        err := filter.ProcessFilter(msg)
        assert.NoError(t, err)
    })

    t.Run("Single_Filter_Success", func(t *testing.T) {
        filter := &MessageFilter{}
        processed := false
        
        filter.AddFilterFunc(func(msg *model.Message) error {
            processed = true
            return nil
        })

        msg := &model.Message{}
        err := filter.ProcessFilter(msg)
        
        assert.NoError(t, err)
        assert.True(t, processed)
    })

    t.Run("Multiple_Filters_Success", func(t *testing.T) {
        filter := &MessageFilter{}
        count := 0
        
        filter.AddFilterFunc(func(msg *model.Message) error {
            count++
            return nil
        })
        
        filter.AddFilterFunc(func(msg *model.Message) error {
            count++
            return nil
        })

        msg := &model.Message{}
        err := filter.ProcessFilter(msg)
        
        assert.NoError(t, err)
        assert.Equal(t, 2, count)
    })

    t.Run("Filter_Error", func(t *testing.T) {
        filter := &MessageFilter{}
        expectedErr := errors.New("filter error")
        
        filter.AddFilterFunc(func(msg *model.Message) error {
            return expectedErr
        })

        msg := &model.Message{
            Header: model.MessageHeader{
                ID: "test-msg",
            },
        }
        
        err := filter.ProcessFilter(msg)
        assert.Error(t, err)
        assert.Equal(t, expectedErr, err)
    })
}