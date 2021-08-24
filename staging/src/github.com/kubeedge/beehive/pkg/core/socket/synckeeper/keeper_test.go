package synckeeper

import (
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// TestNewKeeper test new keeper
func TestNewKeeper(t *testing.T) {
	keeper := NewKeeper()
	message := model.NewMessage("").SetRoute("source", "dest").FillBody("hello")
	ch := keeper.AddKeepChannel(message.GetID())
	go func() {
		err := keeper.SendToKeepChannel(*message.NewRespByMessage(message, "response"))
		if err != nil {
			t.Fatalf("failed to send to keeper")
		}
	}()

	select {
	case msg := <-ch:
		if !keeper.IsSyncResponse(msg.GetParentID()) {
			t.Fatalf("bad parent id")
		}
	case <-time.After(time.Second):
		t.Fatalf("time out")
	}
	keeper.DeleteKeepChannel(message.GetID())
}
