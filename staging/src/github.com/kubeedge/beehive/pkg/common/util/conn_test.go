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
