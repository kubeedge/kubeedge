package context

import (
	"fmt"
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestSendSync(t *testing.T) {
	InitContext(MsgCtxTypeChannel)
	AddModule("test_src")
	messsage := model.NewMessage("")
	messsage.Content = "hello"

	go func() {
		resp, err := SendSync("test_dest", *messsage, 5*time.Second)
		fmt.Printf("resp: %v, error: %v\n", resp, err)
	}()

	msg, err := Receive("test_dest")
	fmt.Printf("receive msg: %v, error: %v\n", msg, err)
	resp := msg.NewRespByMessage(&msg, "how are you")
	SendResp(*resp)

	time.Sleep(5 * time.Second)
}
