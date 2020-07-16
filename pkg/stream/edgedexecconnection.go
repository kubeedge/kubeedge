package stream

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"k8s.io/klog"

	machineryspdy "k8s.io/apimachinery/pkg/util/httpstream/spdy"
)

type EdgedExecConnection struct {
	MessID   uint64        // message id
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
	return fmt.Sprintf("EDGE_Exec_CONNECTOR Message MessageID %v", e.MessID)
}

func (e *EdgedExecConnection) CacheTunnelMessage(msg *Message) {
	e.ReadChan <- msg
}

func (e *EdgedExecConnection) Serve(tunnel SafeWriteTunneler) error {
	upgrade := machineryspdy.NewSpdyRoundTripper(nil, true, false)

	req, err := http.NewRequest(e.Method, e.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header = e.Header

	klog.Infof("Exec Dial request:%++v", *req)

	con, err := upgrade.Dial(req)
	if err != nil {
		klog.Errorf("%s Dailer error %v", e.String(), err)
		return err
	}
	defer con.Close()
	klog.Infof("Exec Dial successfully ...")
	stop := make(chan struct{})
	go func() {
		for mess := range e.ReadChan {
			switch mess.MessageType {
			case MessageTypeRemoveConnect:
				klog.Infof("%s receive remove client id %v", e.String(), mess.ConnectID)
				close(stop)
				return
			case MessageTypeData:
				klog.Infof("TODO ######## %s receive exec %v data ", e.String())
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

	scan := bufio.NewScanner(con)

	for scan.Scan() {
		select {
		case <-stop:
			klog.Infof("receive stop single, so stop exec scan ...")
			return nil
		default:
		}
		msg := NewMessage(e.MessID, MessageTypeData, scan.Bytes())
		err := msg.WriteTo(tunnel)
		if err != nil {
			klog.Errorf("%v write tunnel message %v error", e.String(), msg)
			return err
		}
		klog.Infof("%v write exec data %v", e.String(), string(scan.Bytes()))
	}

	return nil
}

var _ EdgedConnection = &EdgedExecConnection{}
