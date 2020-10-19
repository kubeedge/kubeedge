package lane

import (
	"io"
	"time"

	"k8s.io/klog"

	"github.com/gorilla/websocket"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/packer"
	"github.com/kubeedge/viaduct/pkg/translator"
)

type WSLane struct {
	writeDeadline time.Time
	readDeadline  time.Time
	conn          *websocket.Conn
}

func NewWSLane(van interface{}) *WSLane {
	if wsConn, ok := van.(*websocket.Conn); ok {
		return &WSLane{conn: wsConn}
	}
	klog.Error("oops! bad type of van")
	return nil
}

func (l *WSLane) Read(p []byte) (int, error) {
	_, msgData, err := l.conn.ReadMessage()
	if err != nil {
		if err != io.EOF {
			klog.Errorf("read message error(%+v)", err)
		}
		return len(msgData), err
	}
	p = append(p[:0], msgData...)
	return len(msgData), err
}

func (l *WSLane) ReadMessage(msg *model.Message) error {
	rawData, err := packer.NewReader(l).Read()
	if err != nil {
		return err
	}

	err = translator.NewTran().Decode(rawData, msg)
	if err != nil {
		klog.Error("failed to decode message")
		return err
	}

	return nil
}

func (l *WSLane) Write(p []byte) (int, error) {
	err := l.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		klog.Errorf("write websocket message error(%+v)", err)
		return len(p), err
	}
	return len(p), err
}

func (l *WSLane) WriteMessage(msg *model.Message) error {
	rawData, err := translator.NewTran().Encode(msg)
	if err != nil {
		klog.Error("failed to encode message")
		return err
	}

	_, err = packer.NewWriter(l).Write(rawData)
	return err
}

func (l *WSLane) SetReadDeadline(t time.Time) error {
	l.readDeadline = t
	return l.conn.SetReadDeadline(t)
}

func (l *WSLane) SetWriteDeadline(t time.Time) error {
	l.writeDeadline = t
	return l.conn.SetWriteDeadline(t)
}
