/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
