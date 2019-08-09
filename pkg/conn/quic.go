package conn

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/comm"
	"github.com/kubeedge/viaduct/pkg/fifo"
	"github.com/kubeedge/viaduct/pkg/keeper"
	"github.com/kubeedge/viaduct/pkg/lane"
	"github.com/kubeedge/viaduct/pkg/mux"
	"github.com/kubeedge/viaduct/pkg/smgr"
	"github.com/lucas-clemente/quic-go"
)

var (
	// TODO:
	autoFree = false
)

// the connection based on quic protocol
type QuicConnection struct {
	writeDeadline time.Time
	readDeadline  time.Time
	session       smgr.Session
	handler       mux.Handler
	ctrlLan       lane.Lane
	state         *ConnectionState
	streamManager *smgr.StreamManager
	consumer      io.Writer
	connUse       api.UseType
	syncKeeper    *keeper.SyncKeeper
	messageFifo   *fifo.MessageFifo
	autoRoute     bool
}

// new quic connection
func NewQuicConn(options *ConnectionOptions) *QuicConnection {
	quicSession := options.Base.(quic.Session)
	return &QuicConnection{
		session:       smgr.Session{quicSession},
		handler:       options.Handler,
		ctrlLan:       options.CtrlLane.(lane.Lane),
		state:         options.State,
		connUse:       options.ConnUse,
		syncKeeper:    keeper.NewSyncKeeper(),
		consumer:      options.Consumer,
		autoRoute:     options.AutoRoute,
		messageFifo:   fifo.NewMessageFifo(),
		streamManager: smgr.NewStreamManager(smgr.NumStreamsMax, autoFree, quicSession),
	}
}

// process header message
func (conn *QuicConnection) headerMessage(msg *model.Message) error {
	headers := make(http.Header)
	err := json.Unmarshal(msg.GetContent().([]byte), &headers)
	if err != nil {
		klog.Errorf("failed to unmarshal header, error: %+v", err)
		return err
	}
	conn.state.Headers = headers
	return nil
}

// process control messages
func (conn *QuicConnection) processControlMessage(msg *model.Message) error {
	switch msg.GetOperation() {
	case comm.ControlTypeConfig:
	case comm.ControlTypePing:
	case comm.ControlTypePong:
	}
	return nil
}

// read control message from control lan and process
func (conn *QuicConnection) serveControlLan() {
	var msg model.Message
	for {
		// read control message
		err := conn.ctrlLan.ReadMessage(&msg)
		if err != nil {
			klog.Error("failed read control message")
			return
		}

		// process control message
		result := comm.RespTypeAck
		err = conn.processControlMessage(&msg)
		if err != nil {
			result = comm.RespTypeNack
		}

		// feedback the response
		resp := msg.NewRespByMessage(&msg, result)
		err = conn.ctrlLan.WriteMessage(resp)
		if err != nil {
			klog.Errorf("failed to send response back, error:%+v", err)
			return
		}
	}
}

// ServeSession accept steams from remote peer
// then, receive messages from the steam
func (conn *QuicConnection) serveSession() {
	for {
		stream, err := conn.session.AcceptStream()
		if err != nil {
			// close the session
			klog.Warningf("accept stream error(%+v) or the session has been closed",
				err)
			// close local session
			conn.Close()
			return
		}
		conn.streamManager.AddStream(stream)
		// auto route to mux or raw data consumer
		conn.dispatch(stream)
	}
}

// dispatch the message
func (conn *QuicConnection) dispatch(stream *smgr.Stream) {
	if stream.UseType == api.UseTypeMessage {
		go conn.handleMessage(stream)
	} else if stream.UseType == api.UseTypeStream {
		go conn.handleRawData(stream)
	} else {
		klog.Warningf("bad stream use type(%s), ignore", stream.UseType)
	}
}

// ServeConn start control lan and session loop
func (conn *QuicConnection) ServeConn() {
	go conn.serveControlLan()
	conn.serveSession()
}

func (conn *QuicConnection) Read(raw []byte) (int, error) {
	// get stream from pool
	stream, err := conn.streamManager.GetStream(api.UseTypeStream, false, nil)
	if err != nil {
		// close the session
		klog.Warningf("accept stream error(%+v) or the session has been closed", err)
		return 0, err
	}
	defer conn.streamManager.ReleaseStream(api.UseTypeStream, stream)
	return lane.NewLane(api.ProtocolTypeQuic, stream).Read(raw)
}

// open stream and dispatch message for the new stream
func (conn *QuicConnection) openStreamSync(streamUse api.UseType, autoDispatch bool) (*smgr.Stream, error) {
	stream, err := conn.session.OpenStreamSync(streamUse)
	if err != nil {
		klog.Errorf("failed to open stream, error: %+v", err)
		return stream, err
	}
	// start dispatch for the new stream
	if autoDispatch {
		conn.dispatch(stream)
	}
	return stream, err
}

// accept stream and dispatch message for the new stream
func (conn *QuicConnection) acceptStream(streamUse api.UseType, autoDispatch bool) (*smgr.Stream, error) {
	stream, err := conn.session.AcceptStream()
	if err != nil {
		klog.Errorf("failed to accept stream, error: %+v", err)
		return stream, err
	}
	// start dispatch for the new stream
	if autoDispatch {
		conn.dispatch(stream)
	}
	return stream, err
}

// Write write raw data into stream
func (conn *QuicConnection) Write(raw []byte) (int, error) {
	stream, err := conn.streamManager.GetStream(api.UseTypeStream, false, conn.openStreamSync)
	if err != nil {
		klog.Errorf("failed to acquire stream sync, error:%+v", err)
		return 0, err
	}
	defer conn.streamManager.ReleaseStream(api.UseTypeStream, stream)
	return lane.NewLane(api.ProtocolTypeQuic, stream).Write(raw)
}

func (conn *QuicConnection) handleRawData(stream *smgr.Stream) {
	if conn.consumer == nil {
		klog.Warning("bad raw data consumer")
		return
	}

	if !conn.autoRoute {
		return
	}

	_, err := io.Copy(conn.consumer, lane.NewLane(api.ProtocolTypeQuic, stream.Stream))
	if err != nil {
		klog.Errorf("failed to copy data, error: %+v", err)
		return
	}
}

// read message from stream and route the message to mux
func (conn *QuicConnection) handleMessage(stream *smgr.Stream) {
	msg := &model.Message{}
	for {
		err := lane.NewLane(api.ProtocolTypeQuic, stream.Stream).ReadMessage(msg)
		if err != nil {
			if err != io.EOF {
				klog.Errorf("failed to read message, error: %+v", err)
			}
			conn.streamManager.FreeStream(stream)
			return
		}

		// to check whether the message is a response or not
		if matched := conn.syncKeeper.MatchAndNotify(*msg); matched {
			continue
		}

		// put the messages into fifo and wait for reading
		if !conn.autoRoute {
			conn.messageFifo.Put(msg)
			continue
		}

		// user do not set  message handle, use the default mux
		if conn.handler == nil {
			// use default mux
			conn.handler = mux.MuxDefault
		}
		conn.handler.ServeConn(&mux.MessageRequest{
			Header:  conn.state.Headers,
			Message: msg,
		}, &responseWriter{
			Type: api.ProtocolTypeQuic,
			Van:  stream.Stream,
		})
	}
}

// Close will cancel write and read
// close the session
func (conn *QuicConnection) Close() error {
	conn.state.State = api.StatDisconnected
	conn.streamManager.Destroy()
	return conn.session.Close()
}

// WriteMessageSync write sync message
// please set write deadline before WriteMessageSync called
func (conn *QuicConnection) WriteMessageSync(msg *model.Message) (*model.Message, error) {
	if conn.session.Sess == nil {
		klog.Error("bad connection session")
		return nil, fmt.Errorf("bad connection session")
	}

	stream, err := conn.streamManager.GetStream(api.UseTypeMessage, true, conn.openStreamSync)
	if err != nil {
		klog.Errorf("failed to acquire stream sync, error:%+v", err)
		return nil, fmt.Errorf("failed to acquire stream sync, error:%+v", err)
	}
	defer conn.streamManager.ReleaseStream(api.UseTypeMessage, stream)

	lane := lane.NewLane(api.ProtocolTypeQuic, stream)
	lane.SetWriteDeadline(conn.writeDeadline)
	msg.Header.Sync = true
	err = lane.WriteMessage(msg)
	if err != nil {
		return nil, err
	}

	// receive response
	response, err := conn.syncKeeper.WaitResponse(msg, conn.writeDeadline)
	return &response, nil
}

// WriteMessageAsync send async message
func (conn *QuicConnection) WriteMessageAsync(msg *model.Message) error {
	if conn.session.Sess == nil {
		klog.Error("bad connection session")
		return fmt.Errorf("bad connection session")
	}

	stream, err := conn.streamManager.GetStream(api.UseTypeMessage, true, conn.openStreamSync)
	if err != nil {
		klog.Errorf("failed to acquire stream sync, error:%+v", err)
		return fmt.Errorf("failed to acquire stream sync, error:%+v", err)
	}
	defer conn.streamManager.ReleaseStream(api.UseTypeMessage, stream)

	lane := lane.NewLane(api.ProtocolTypeQuic, stream)
	lane.SetWriteDeadline(conn.writeDeadline)
	msg.Header.Sync = false
	return lane.WriteMessage(msg)
}

// ReadMessage read message from fifo
// it will blocked when no message received
func (conn *QuicConnection) ReadMessage(msg *model.Message) error {
	return conn.messageFifo.Get(msg)
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
	return conn.session.Sess.RemoteAddr()
}

func (conn *QuicConnection) LocalAddr() net.Addr {
	return conn.session.Sess.LocalAddr()
}

func (conn *QuicConnection) ConnectionState() ConnectionState {
	conn.state.PeerCertificates = conn.session.Sess.ConnectionState().PeerCertificates
	return *conn.state
}
