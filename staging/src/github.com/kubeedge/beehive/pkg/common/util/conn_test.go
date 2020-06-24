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

package util

import (
	"fmt"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	us := NewUnixSocket("/tmp/us.socket")
	us.SetContextHandler(func(context string) string {
		fmt.Println(context)
		now := "response from server: " + time.Now().String()
		return now
	})
	go us.StartServer()
	time.Sleep(time.Second * 10)
}

func TestClient(t *testing.T) {
	us := NewUnixSocket("/tmp/us.socket")
	r := us.Connect()
	time.Sleep(time.Second * 10)
	res := us.Send(r, "zhangqi test")
	fmt.Println("===============" + res)
}
