package lane

import (
	"io"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
)

type WSLaneWithoutPack struct {
	writeDeadline time.Time
	readDeadline  time.Time
	conn          *websocket.Conn
}

func NewWSLaneWithoutPack(van interface{}) *WSLaneWithoutPack {
	if wsConn, ok := van.(*websocket.Conn); ok {
		return &WSLaneWithoutPack{conn: wsConn}
	}
	log.LOGGER.Errorf("oops! bad type of van")
	return nil
}

func (l *WSLaneWithoutPack) Read(p []byte) (int, error) {
	_, msgData, err := l.conn.ReadMessage()
	if err != nil {
		if err != io.EOF {
			log.LOGGER.Errorf("read message error(%+v)", err)
		}
		return len(msgData), err
	}
	p = append(p[:0], msgData...)
	return len(msgData), err
}

func (l *WSLaneWithoutPack) ReadMessage(msg *model.Message) error {
	return l.conn.ReadJSON(msg)
}

func (l *WSLaneWithoutPack) Write(p []byte) (int, error) {
	err := l.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		log.LOGGER.Errorf("write websocket message error(%+v)", err)
		return len(p), err
	}
	return len(p), err
}

func (l *WSLaneWithoutPack) WriteMessage(msg *model.Message) error {
	return l.conn.WriteJSON(msg)
}

func (l *WSLaneWithoutPack) SetReadDeadline(t time.Time) error {
	l.readDeadline = t
	return l.conn.SetReadDeadline(t)
}

func (l *WSLaneWithoutPack) SetWriteDeadline(t time.Time) error {
	l.writeDeadline = t
	return l.conn.SetWriteDeadline(t)
}
