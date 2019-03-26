package conn

import (
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/lane"
	"github.com/kubeedge/viaduct/pkg/mux"
)

type WSConnection struct {
	WriteDeadline time.Time
	ReadDeadline  time.Time

	wsConn  *websocket.Conn
	handler mux.Handler
}

func NewWSConn(options *ConnectionOptions) *WSConnection {
	return &WSConnection{
		wsConn:  options.Base.(*websocket.Conn),
		handler: options.Handler,
	}
}

func (conn *WSConnection) ServeConn() {
	go conn.handleMessage()
}

func (conn *WSConnection) handleMessage() {
	msg := &model.Message{}
	for {
		err := lane.NewLane(api.ProtocolTypeWS, conn.wsConn).ReadMessage(msg)
		if err != nil {
			if err != io.EOF {
				log.LOGGER.Errorf("failed to read message, error: %+v", err)
			}
			return
		}
		if conn.handler == nil {
			// use default mux
			conn.handler = mux.MuxDefault
		}
		conn.handler.ServeConn(msg, &responseWriter{
			Type: api.ProtocolTypeWS,
			Van:  conn.wsConn,
		})
	}
}

func (conn *WSConnection) SetReadDeadline(t time.Time) error {
	conn.ReadDeadline = t
	return nil
}

func (conn *WSConnection) SetWriteDeadline(t time.Time) error {
	conn.WriteDeadline = t
	return nil
}

func (conn *WSConnection) WriteMessageAsync(msg *model.Message) error {
	lane := lane.NewLane(api.ProtocolTypeWS, conn.wsConn)
	lane.SetReadDeadline(conn.WriteDeadline)
	return lane.WriteMessage(msg)
}

func (conn *WSConnection) WriteMessageSync(msg *model.Message) (*model.Message, error) {
	lane := lane.NewLane(api.ProtocolTypeWS, conn.wsConn)
	// send msg
	lane.SetWriteDeadline(conn.WriteDeadline)
	err := lane.WriteMessage(msg)
	if err != nil {
		log.LOGGER.Errorf("write message error(%+v)", err)
		return nil, err
	}

	//receive response
	response := model.Message{}
	lane.SetReadDeadline(conn.WriteDeadline)
	err = lane.ReadMessage(&response)
	if err != nil {
		log.LOGGER.Errorf("receive response error(%+v)", err)
		return nil, err
	}
	return nil, nil
}

func (conn *WSConnection) ReadMessage(msg *model.Message) error {
	lane := lane.NewLane(api.ProtocolTypeWS, conn.wsConn)
	lane.SetReadDeadline(conn.ReadDeadline)
	return lane.ReadMessage(msg)
}

func (conn *WSConnection) RemoteAddr() net.Addr {
	return conn.wsConn.RemoteAddr()
}

func (conn *WSConnection) LocalAddr() net.Addr {
	return conn.wsConn.LocalAddr()
}

func (conn *WSConnection) Close() error {
	return conn.wsConn.Close()
}

// get connection state
// TODO:
func (conn *WSConnection) ConnectionState() ConnectionState {
	return ConnectionState{}
}
