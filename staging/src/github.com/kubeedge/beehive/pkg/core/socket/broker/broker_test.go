package broker

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/beehive/pkg/core/socket/wrapper"
)

func serveSocket(socketType, address string, handle func(conn wrapper.Conn)) error {
	if strings.Contains(socketType, "unix") {
		os.Remove(address)
	}

	listener, err := net.Listen(socketType, address)
	defer func() {
		if listener != nil {
			err := listener.Close()
			if err != nil {
				klog.Errorf("listener close err %+v", err)
			}
		}
	}()
	if err != nil {
		klog.Errorf("failed to listen to unix domain socket with error %+v", err)
		return fmt.Errorf("failed to listen to unix domain socket, error: %+v ", err)
	}
	klog.Infof("Listening on addr: %s", listener.Addr().String())

	conn, err := listener.Accept()
	klog.Infof("Connected from %s", conn.LocalAddr().String())
	if err != nil {
		klog.Errorf("failed to accept with error %+v", err)
		return fmt.Errorf("failed to accept, error: %+v", err)
	}

	wrapper := wrapper.NewWrapper(socketType, conn, 10240)
	handle(wrapper)
	err = conn.Close()

	return err
}

// SocketConnect socket connect
func SocketConnect(opts ConnectOptions) (interface{}, error) {
	conn, err := net.Dial(opts.MessageType, opts.Address)
	if err != nil {
		klog.Errorf("failed to dail addrs %s, error:%+v", opts.Address, err)
		return nil, err
	}
	return conn, nil
}

// TestMessageBroker_SendReceive test message broker_ send receive
func TestMessageBroker_SendReceive(t *testing.T) {
	broker := NewRemoteBroker()
	stopChan := make(chan struct{})
	handle := func(conn wrapper.Conn) {
		message, err := broker.Receive(conn)
		if err != nil {
			t.Fatalf("failed to receive message, error: %+v", err)
		}
		fmt.Printf("recive message: %+v", message)
		if message.GetContent() != "hello" {
			t.Fatalf("bad message content")
		}
		stopChan <- struct{}{}
	}
	go func() {
		err := serveSocket("tcp", "127.0.0.1:1234", handle)
		if err != nil {
			t.Fatalf("failed to serve socket")
		}
	}()

	time.Sleep(1 * time.Second)
	brokerClient := NewRemoteBroker()

	opts := ConnectOptions{
		Address:     "127.0.0.1:1234",
		MessageType: "tcp",
		BufferSize:  10240,
	}
	conn := brokerClient.Connect(opts, SocketConnect)
	if conn == nil {
		t.Fatalf("failed to connect tcp")
	}
	brokerClient.Send(conn, *model.NewMessage("").FillBody("hello"))

	select {
	case _, ok := <-time.After(syncMessageTimeoutDefault):
		if ok {
			t.Fatalf("time out ti recive message")
		}
	case _, ok := <-stopChan:
		if ok {
			klog.Warningf("channel stopped")
		}
	}
	conn.Close()
}

// TestMessageBroker_SendSync test message broker_ send sync
func TestMessageBroker_SendSync(t *testing.T) {
	brokerServer := NewRemoteBroker()
	stopTChan := make(chan struct{})
	handle := func(conn wrapper.Conn) {
		message, err := brokerServer.Receive(conn)
		if err != nil {
			t.Fatalf("failed to receive message, error: %+v", err)
		}
		if message.GetContent() != "hello" {
			t.Fatalf("bad message content")
		}
		resp := message.NewRespByMessage(&message, "hello_response")
		brokerServer.Send(conn, *resp)
	}
	go func() {
		err := serveSocket("tcp", "127.0.0.1:1234", handle)
		if err != nil {
			t.Fatalf("failed to serve socket")
		}
	}()

	time.Sleep(1 * time.Second)
	brokerClient := NewRemoteBroker()
	opts := ConnectOptions{
		Address:     "127.0.0.1:1234",
		MessageType: "tcp",
		BufferSize:  10240,
	}
	conn := brokerClient.Connect(opts, SocketConnect)
	if conn == nil {
		t.Fatalf("failed to connect tcp")
	}

	go func() {
		brokerClient.Receive(conn)
	}()

	go func() {
		resp, err := brokerClient.SendSync(conn, *model.NewMessage("").
			SetRoute("source", "dest").FillBody("hello"), 0)
		if err != nil {
			t.Fatalf("failed to send sync message, error: %+v", err)
		}
		if resp.GetContent() != "hello_response" {
			t.Fatalf("unexpected receive message")
		}
		stopTChan <- struct{}{}
	}()

	select {
	case _, ok := <-time.After(syncMessageTimeoutDefault):
		if ok {
			t.Fatalf("time out to send sync")
		}
	case _, ok := <-stopTChan:
		if ok {
			klog.Warningf("channel stopped")
		}
	}
	conn.Close()
}
