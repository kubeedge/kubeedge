/*
Copyright 2023 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

type EdgedAttachConnection struct {
	ReadChan chan *Message `json:"-"`
	Stop     chan struct{} `json:"-"`
	MessID   uint64
	URL      url.URL     `json:"url"`
	Header   http.Header `json:"header"`
	Method   string      `json:"method"`
}

func (ah *EdgedAttachConnection) CreateConnectMessage() (*Message, error) {
	data, err := json.Marshal(ah)
	if err != nil {
		return nil, err
	}
	return NewMessage(ah.MessID, MessageTypeAttachConnect, data), nil
}

func (ah *EdgedAttachConnection) GetMessageID() uint64 {
	return ah.MessID
}

func (ah *EdgedAttachConnection) String() string {
	return fmt.Sprintf("EDGE_ATTACH_CONNECTOR Message MessageID %v", ah.MessID)
}

func (ah *EdgedAttachConnection) CacheTunnelMessage(msg *Message) {
	ah.ReadChan <- msg
}

func (ah *EdgedAttachConnection) CloseReadChannel() {
	close(ah.ReadChan)
}

func (ah *EdgedAttachConnection) CleanChannel() {
	for {
		select {
		case <-ah.Stop:
		default:
			return
		}
	}
}

func (ah *EdgedAttachConnection) receiveFromCloudStream(con net.Conn, stop chan struct{}) {
	for message := range ah.ReadChan {
		switch message.MessageType {
		case MessageTypeRemoveConnect:
			klog.V(6).Infof("%s receive remove client id %v", ah.String(), message.ConnectID)
			stop <- struct{}{}
		case MessageTypeData:
			_, err := con.Write(message.Data)
			klog.V(6).Infof("%s receive attach %v data ", ah.String(), message.Data)
			if err != nil {
				klog.Errorf("failed to write, err: %v", err)
			}
		}
	}
	klog.V(6).Infof("%s read channel closed", ah.String())
}

func (ah *EdgedAttachConnection) write2CloudStream(tunnel SafeWriteTunneler, con net.Conn, stop chan struct{}) {
	defer func() {
		stop <- struct{}{}
	}()

	var data [256]byte
	for {
		n, err := con.Read(data[:])
		if err != nil {
			if !errors.Is(err, io.EOF) {
				klog.Errorf("%v failed to read attach data, err:%v", ah.String(), err)
			}
			return
		}
		msg := NewMessage(ah.MessID, MessageTypeData, data[:n])
		if err := tunnel.WriteMessage(msg); err != nil {
			klog.Errorf("%v failed to write to tunnel, msg: %+v, err: %v", ah.String(), msg, err)
			return
		}
		klog.V(6).Infof("%v write attach data %v", ah.String(), data[:n])
	}
}

func (ah *EdgedAttachConnection) Serve(tunnel SafeWriteTunneler) error {
	tripper, err := spdy.NewRoundTripper(nil)
	if err != nil {
		return fmt.Errorf("failed to creates a new tripper, err: %v", err)
	}
	req, err := http.NewRequest(ah.Method, ah.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create attach request, err: %v", err)
	}
	req.Header = ah.Header
	con, err := tripper.Dial(req)
	if err != nil {
		klog.Errorf("failed to dial, err: %v", err)
		return err
	}
	defer con.Close()

	go ah.receiveFromCloudStream(con, ah.Stop)

	defer func() {
		for retry := 0; retry < 3; retry++ {
			msg := NewMessage(ah.MessID, MessageTypeRemoveConnect, nil)
			if err := tunnel.WriteMessage(msg); err != nil {
				klog.Errorf("%v send %s message error %v", ah, msg.MessageType, err)
			} else {
				break
			}
		}
	}()

	go ah.write2CloudStream(tunnel, con, ah.Stop)

	<-ah.Stop
	klog.V(6).Infof("receive stop signal, so stop attach scan ...")
	return nil
}

var _ EdgedConnection = &EdgedAttachConnection{}
