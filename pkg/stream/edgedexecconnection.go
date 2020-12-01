package stream

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/klog/v2"
)

type EdgedExecConnection struct {
	MessID   uint64
	URL      url.URL       `json:"url"`
	Header   http.Header   `json:"header"`
	Method   string        `json:"method"`
	ReadChan chan *Message `json:"-"`
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

type responder struct{}

func (r *responder) Error(w http.ResponseWriter, req *http.Request, err error) {
	klog.Errorf("failed to proxy request: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (e *EdgedExecConnection) Serve(tunnel SafeWriteTunneler) error {
	tripper := spdy.NewRoundTripper(nil, true, false)
	req, err := http.NewRequest(e.Method, e.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create exec request, err: %v", err)
	}
	req.Header = e.Header

	//handler := proxy.NewUpgradeAwareHandler(&e.URL, nil, false, true, &responder{})

	con, err := tripper.Dial(req)
	if err != nil {
		klog.Errorf("failed to dial, err: %v", err)
		return err
	}
	defer con.Close()

	stop := make(chan struct{})

	go func() {
		for message := range e.ReadChan {
			switch message.MessageType {
			case MessageTypeRemoveConnect:
				klog.V(6).Infof("%s receive remove client id %v", e.String(), message.ConnectID)
				close(stop)
				return
			case MessageTypeData:
				_, err := con.Write(message.Data)
				klog.V(6).Infof("%s receive exec %v data ", e.String(), message.Data)
				if err != nil {
					klog.Errorf("failed to write, err: %v", err)
				}
			}
		}
	}()

	defer func() {
		for retry := 0; retry < 3; retry++ {
			msg := NewMessage(e.MessID, MessageTypeRemoveConnect, nil)
			if err := msg.WriteTo(tunnel); err != nil {
				klog.Errorf("%v send %s message error %v", e, msg.MessageType, err)
			} else {
				break
			}
		}
	}()

	for {
		select {
		case <-stop:
			klog.V(6).Infof("receive stop single, so stop exec scan ...")
			return nil
		default:
		}
		data := make([]byte, 256)

		n, err := con.Read(data)
		if err != nil {
			if err != io.EOF {
				klog.Errorf("%v failed to write exec data, err:%v", e.String(), err)
			}
			break
		}
		if n <= 0 {
			continue
		}
		msg := NewMessage(e.MessID, MessageTypeData, data[:n])
		if err := msg.WriteTo(tunnel); err != nil {
			klog.Errorf("%v failed to write to tunnel, msg: %+v, err: %v", e.String(), msg, err)
			return err
		}
		klog.V(6).Infof("%v write exec data %v", e.String(), data[:n])
	}
	return nil
}

var _ EdgedConnection = &EdgedExecConnection{}
