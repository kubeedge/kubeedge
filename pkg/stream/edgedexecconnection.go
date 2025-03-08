package stream

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"k8s.io/klog/v2"
)

type EdgedExecConnection struct {
	BaseEdgedConnection `json:",inline"`
}

func (e *EdgedExecConnection) CreateConnectMessage() (*Message, error) {
	return e.createConnectMessage(MessageTypeExecConnect, e)
}

func (e *EdgedExecConnection) String() string {
	return fmt.Sprintf("EDGE_EXEC_CONNECTOR Message MessageID %v", e.MessID)
}

type responder struct{}

func (r *responder) Error(w http.ResponseWriter, _ *http.Request, err error) {
	klog.Errorf("failed to proxy request: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (e *EdgedExecConnection) receiveFromCloudStream(con net.Conn) {
	for message := range e.ReadChan {
		switch message.MessageType {
		case MessageTypeRemoveConnect:
			klog.V(6).Infof("%s receive remove client id %v", e.String(), message.ConnectID)
			e.Stop <- struct{}{}

		case MessageTypeData:
			_, err := con.Write(message.Data)
			klog.V(6).Infof("%s receive exec %v data ", e.String(), message.Data)
			if err != nil {
				klog.Errorf("failed to write, err: %v", err)
			}
		}
	}
	klog.V(2).Infof("%s read channel closed", e.String())
}

func (e *EdgedExecConnection) write2CloudStream(tunnel SafeWriteTunneler, con net.Conn) {
	defer func() {
		e.Stop <- struct{}{}
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
	return e.serveByRoundTripper(tunnel, roundTripperCustomization{
		name:                   e.String(),
		receiveFromCloudStream: e.receiveFromCloudStream,
		write2CloudStream:      e.write2CloudStream,
	})
}

var _ EdgedConnection = &EdgedExecConnection{}
