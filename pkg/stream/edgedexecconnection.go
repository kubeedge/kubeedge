package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/klog/v2"
)

type EdgedExecConnection struct {
	ReadChan chan *Message `json:"-"`
	Stop     chan struct{} `json:"-"`
	MessID   uint64
	URL      url.URL     `json:"url"`
	Header   http.Header `json:"header"`
	Method   string      `json:"method"`
}

func (e *EdgedExecConnection) CreateConnectMessage() (*Message, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return NewMessage(e.MessID, MessageTypeExecConnect, data), nil
}

func (e *EdgedExecConnection) GetMessageID() uint64 {
	return e.MessID
}

func (e *EdgedExecConnection) String() string {
	return fmt.Sprintf("EDGE_EXEC_CONNECTOR Message MessageID %v", e.MessID)
}

func (e *EdgedExecConnection) CacheTunnelMessage(msg *Message) {
	e.ReadChan <- msg
}

func (e *EdgedExecConnection) CloseReadChannel() {
	close(e.ReadChan)
}

func (e *EdgedExecConnection) CleanChannel() {
	for {
		select {
		case <-e.Stop:
		default:
			return
		}
	}
}

type responder struct{}

func (r *responder) Error(w http.ResponseWriter, req *http.Request, err error) {
	klog.Errorf("failed to proxy request: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (e *EdgedExecConnection) receiveFromCloudStream(con net.Conn, stop chan struct{}) {
	for message := range e.ReadChan {
		switch message.MessageType {
		case MessageTypeRemoveConnect:
			klog.V(6).Infof("%s receive remove client id %v", e.String(), message.ConnectID)
			stop <- struct{}{}
		case MessageTypeData:
			_, err := con.Write(message.Data)
			klog.V(6).Infof("%s receive exec %v data ", e.String(), message.Data)
			if err != nil {
				klog.Errorf("failed to write, err: %v", err)
			}
		}
	}
	klog.V(6).Infof("%s read channel closed", e.String())
}

func (e *EdgedExecConnection) write2CloudStream(tunnel SafeWriteTunneler, con net.Conn, stop chan struct{}) {
	defer func() {
		stop <- struct{}{}
	}()

	var data [256]byte
	for {
		n, err := con.Read(data[:])
		if err != nil {
			if !errors.Is(err, io.EOF) {
				klog.Errorf("%v failed to read exec data, err:%v", e.String(), err)
			}
			return
		}
		msg := NewMessage(e.MessID, MessageTypeData, data[:n])
		if err := tunnel.WriteMessage(msg); err != nil {
			klog.Errorf("%v failed to write to tunnel, msg: %+v, err: %v", e.String(), msg, err)
			return
		}
		klog.V(6).Infof("%v write exec data %v", e.String(), data[:n])
	}
}

func (e *EdgedExecConnection) Serve(tunnel SafeWriteTunneler) error {
	tripper := spdy.NewRoundTripper(nil, true, false)
	req, err := http.NewRequest(e.Method, e.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create exec request, err: %v", err)
	}
	req.Header = e.Header
	con, err := tripper.Dial(req)
	if err != nil {
		klog.Errorf("failed to dial, err: %v", err)
		return err
	}
	defer con.Close()

	go e.receiveFromCloudStream(con, e.Stop)

	defer func() {
		for retry := 0; retry < 3; retry++ {
			msg := NewMessage(e.MessID, MessageTypeRemoveConnect, nil)
			if err := tunnel.WriteMessage(msg); err != nil {
				klog.Errorf("%v send %s message error %v", e, msg.MessageType, err)
			} else {
				break
			}
		}
	}()

	go e.write2CloudStream(tunnel, con, e.Stop)

	<-e.Stop
	klog.V(6).Infof("receive stop signal, so stop exec scan ...")
	return nil
}

var _ EdgedConnection = &EdgedExecConnection{}
