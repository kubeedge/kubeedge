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

type EdgedPortForwardConnection struct {
	ReadChan chan *Message `json:"-"`
	Stop     chan struct{} `json:"-"`
	MessID   uint64
	URL      url.URL     `json:"url"`
	Header   http.Header `json:"header"`
	Method   string      `json:"method"`
}

func (p *EdgedPortForwardConnection) CreateConnectMessage() (*Message, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return NewMessage(p.MessID, MessageTypePortForwardConnect, data), nil
}

func (p *EdgedPortForwardConnection) GetMessageID() uint64 {
	return p.MessID
}

func (p *EdgedPortForwardConnection) String() string {
	return fmt.Sprintf("EDGE_PORT_FORWARD_CONNECTOR Message MessageID %v", p.MessID)
}

func (p *EdgedPortForwardConnection) CacheTunnelMessage(msg *Message) {
	p.ReadChan <- msg
}

func (p *EdgedPortForwardConnection) CloseReadChannel() {
	close(p.ReadChan)
}

func (p *EdgedPortForwardConnection) CleanChannel() {
	for {
		select {
		case <-p.Stop:
		default:
			return
		}
	}
}

func (p *EdgedPortForwardConnection) receiveFromCloudStream(con net.Conn, stop chan struct{}) {
	for message := range p.ReadChan {
		switch message.MessageType {
		case MessageTypeRemoveConnect:
			klog.Infof("%s receive remove client id %v", p.String(), message.ConnectID)
			stop <- struct{}{}
		case MessageTypeData:
			if _, err := con.Write(message.Data); err != nil {
				klog.Errorf("%s failed to write portForward data, err: %v", p.String(), err)
			}
		}
	}
	klog.V(0).Infof("%s read channel closed", p.String())
}

func (p *EdgedPortForwardConnection) write2CloudStream(tunnel SafeWriteTunneler, con net.Conn, stop chan struct{}) {
	defer func() {
		stop <- struct{}{}
	}()

	var data [256]byte
	for {
		n, err := con.Read(data[:])
		if err != nil {
			if !errors.Is(err, io.EOF) {
				klog.Errorf("%v failed to read portForward data, err:%v", p.String(), err)
			}
			return
		}
		msg := NewMessage(p.MessID, MessageTypeData, data[:n])
		if err := tunnel.WriteMessage(msg); err != nil {
			klog.Errorf("%v failed to write to tunnel, msg: %+v, err: %v", p.String(), msg, err)
			return
		}
	}
}

func (p *EdgedPortForwardConnection) Serve(tunnel SafeWriteTunneler) error {
	tripper, err := spdy.NewRoundTripper(nil)
	if err != nil {
		return fmt.Errorf("failed to create a new tripper, err: %v", err)
	}

	req, err := http.NewRequest(p.Method, p.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create portForward request, err: %v", err)
	}
	req.Header = p.Header

	con, err := tripper.Dial(req)
	if err != nil {
		klog.Errorf("failed to dial portForward, err: %v", err)
		return err
	}
	defer con.Close()

	klog.Infof("successfully connected local portForward endpoint: %s", req.URL.String())

	go p.receiveFromCloudStream(con, p.Stop)

	defer func() {
		for retry := 0; retry < 3; retry++ {
			msg := NewMessage(p.MessID, MessageTypeRemoveConnect, nil)
			if err := tunnel.WriteMessage(msg); err != nil {
				klog.Errorf("%v send %s message error %v", p, msg.MessageType, err)
			} else {
				break
			}
		}
	}()

	go p.write2CloudStream(tunnel, con, p.Stop)

	<-p.Stop
	klog.Infof("receive stop signal, so stop portForward scan ...")
	return nil
}

var _ EdgedConnection = &EdgedPortForwardConnection{}
