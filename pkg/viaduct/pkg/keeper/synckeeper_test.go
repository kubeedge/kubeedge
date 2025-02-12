package keeper

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/kubeedge/beehive/pkg/core/model"
)

func TestSyncKeeper(t *testing.T) {
    t.Run("Initialize", func(t *testing.T) {
        keeper := NewSyncKeeper()
        assert.NotNil(t, keeper)
    })

    t.Run("AddAndMatchChannel", func(t *testing.T) {
        keeper := NewSyncKeeper()
        msgID := "test-1"
        
        channel := keeper.addKeepChannel(msgID)
        assert.NotNil(t, channel)
        
        msg := model.Message{}
        msg.Header.ParentID = msgID
        assert.True(t, keeper.Match(msg))
    })

    t.Run("DeleteChannel", func(t *testing.T) {
        keeper := NewSyncKeeper()
        msgID := "test-1"
        
        keeper.addKeepChannel(msgID)
        keeper.deleteKeepChannel(msgID)
        
        msg := model.Message{}
        msg.Header.ParentID = msgID
        assert.False(t, keeper.Match(msg))
    })

    t.Run("WaitResponse_Success", func(t *testing.T) {
        keeper := NewSyncKeeper()
        msg := &model.Message{
            Header: model.MessageHeader{
                ID: "test-1",
            },
        }
        
        // Simulate response
        go func() {
            time.Sleep(100 * time.Millisecond)
            response := model.Message{
                Header: model.MessageHeader{
                    ParentID: msg.Header.ID,
                    ID:      "response-1",
                },
            }
            keeper.MatchAndNotify(response)
        }()

        deadline := time.Now().Add(time.Second)
        resp, err := keeper.WaitResponse(msg, deadline)
        assert.NoError(t, err)
        assert.Equal(t, "response-1", resp.Header.ID)
    })

    t.Run("WaitResponse_Timeout", func(t *testing.T) {
        keeper := NewSyncKeeper()
        msg := &model.Message{
            Header: model.MessageHeader{
                ID: "test-timeout",
            },
        }
        
        deadline := time.Now().Add(100 * time.Millisecond)
        _, err := keeper.WaitResponse(msg, deadline)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "timeout")
    })

    t.Run("MatchAndNotify_NonExistent", func(t *testing.T) {
        keeper := NewSyncKeeper()
        msg := model.Message{
            Header: model.MessageHeader{
                ParentID: "non-existent",
            },
        }
        
        success := keeper.MatchAndNotify(msg)
        assert.False(t, success)
    })
}