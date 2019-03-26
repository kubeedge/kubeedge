package conn

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/lane"
	"github.com/kubeedge/viaduct/pkg/mux"
	"github.com/lucas-clemente/quic-go"
)

// the connection based on quic protocol
type QuicConnection struct {
	writeDeadline time.Time
	readDeadline  time.Time
	session       quic.Session
	handler       mux.Handler
}

// new quic connection
func NewQuicConn(options *ConnectionOptions) *QuicConnection {
	return &QuicConnection{
		session: options.Base.(quic.Session),
		handler: options.Handler,
	}
}

// ServeConn accept steams from remote peer
// then, receive messages from the steam
func (conn *QuicConnection) ServeConn() {
	for {
		stream, err := conn.session.AcceptStream()
		if err != nil {
			// close the session
			log.LOGGER.Warnf("accept stream error(%+v)"+
				" or the session has been closed", err)
			conn.session.Close()
			return
		}
		go conn.handleMessage(stream)
	}
}

// read message from stream and route the message to mux
func (conn *QuicConnection) handleMessage(stream quic.Stream) {
	msg := &model.Message{}
	for {
		err := lane.NewLane(api.ProtocolTypeQuic, stream).ReadMessage(msg)
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
			Type: api.ProtocolTypeQuic,
			Van:  stream,
		})
	}
}

func (conn *QuicConnection) Close() error {
	return conn.session.Close()
}

func (conn *QuicConnection) WriteMessageSync(msg *model.Message) (*model.Message, error) {
	if conn.session == nil {
		log.LOGGER.Errorf("bad connection session")
		return nil, fmt.Errorf("bad connection session")
	}

	stream, err := conn.session.OpenStreamSync()
	if err != nil {
		log.LOGGER.Errorf("failed to open stream sync, error:%+v", err)
		conn.session.Close()
		return nil, fmt.Errorf("failed to open stream sync, error:%+v", err)
	}
	defer stream.Close()

	lane := lane.NewLane(api.ProtocolTypeQuic, stream)
	lane.SetWriteDeadline(conn.writeDeadline)

	err = lane.WriteMessage(msg)
	if err != nil {
		return nil, err
	}

	// receive response
	response := model.Message{}
	lane.SetReadDeadline(conn.writeDeadline)
	err = lane.ReadMessage(&response)
	if err != nil {
		log.LOGGER.Errorf("receive response failed or timeout, error: %+v", err)
		return &response, err
	}

	return &response, nil
}

func (conn *QuicConnection) WriteMessageAsync(msg *model.Message) error {
	if conn.session == nil {
		log.LOGGER.Errorf("bad connection session")
		return fmt.Errorf("bad connection session")
	}

	stream, err := conn.session.OpenStreamSync()
	if err != nil {
		log.LOGGER.Errorf("failed to open stream sync, error:%+v", err)
		conn.session.Close()
		return fmt.Errorf("failed to open stream sync, error:%+v", err)
	}
	defer stream.Close()

	lane := lane.NewLane(api.ProtocolTypeQuic, stream)
	lane.SetWriteDeadline(conn.writeDeadline)

	return lane.WriteMessage(msg)
}

func (conn *QuicConnection) ReadMessage(msg *model.Message) error {
	if conn.session == nil {
		log.LOGGER.Errorf("bad connection session")
		return fmt.Errorf("bad connection session")
	}

	stream, err := conn.session.AcceptStream()
	if err != nil {
		log.LOGGER.Errorf("failed to accept stream, error: %+v", err)
		return err
	}

	lane := lane.NewLane(api.ProtocolTypeQuic, stream)
	lane.SetReadDeadline(conn.readDeadline)

	return lane.ReadMessage(msg)
}

func (conn *QuicConnection) SetReadDeadline(t time.Time) error {
	conn.readDeadline = t
	return nil
}

func (conn *QuicConnection) SetWriteDeadline(t time.Time) error {
	conn.writeDeadline = t
	return nil
}

func (conn *QuicConnection) RemoteAddr() net.Addr {
	return conn.session.RemoteAddr()
}

func (conn *QuicConnection) LocalAddr() net.Addr {
	return conn.session.LocalAddr()
}

func (conn *QuicConnection) ConnectionState() ConnectionState {
	return ConnectionState{
		PeerCertificates: conn.ConnectionState().PeerCertificates,
	}
}
