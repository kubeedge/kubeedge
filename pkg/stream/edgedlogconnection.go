/*
Copyright 2020 The KubeEdge Authors.

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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"k8s.io/klog/v2"
)

type EdgedLogsConnection struct {
	MessID   uint64        // message id
	URL      url.URL       `json:"url"`
	Header   http.Header   `json:"header"`
	ReadChan chan *Message `json:"-"`
}

func (l *EdgedLogsConnection) GetMessageID() uint64 {
	return l.MessID
}

func (l *EdgedLogsConnection) CacheTunnelMessage(msg *Message) {
	l.ReadChan <- msg
}

func (l *EdgedLogsConnection) CreateConnectMessage() (*Message, error) {
	data, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return NewMessage(l.MessID, MessageTypeLogsConnect, data), nil
}

func (l *EdgedLogsConnection) String() string {
	return fmt.Sprintf("EDGE_LOGS_CONNECTOR Message MessageID %v", l.MessID)
}

func (l *EdgedLogsConnection) Serve(tunnel SafeWriteTunneler) error {
	//connect edged
	client := http.Client{}
	req, err := http.NewRequest("GET", l.URL.String(), nil)
	if err != nil {
		klog.Errorf("create new logs request error %v", err)
		return err
	}
	req.Header = l.Header
	resp, err := client.Do(req)
	if err != nil {
		klog.Errorf("request logs error %v", err)
		return err
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	stop := make(chan struct{})

	go func() {
		for mess := range l.ReadChan {
			if mess.MessageType == MessageTypeRemoveConnect {
				klog.Infof("receive remove client id %v", mess.ConnectID)
				close(stop)
				return
			}
		}
	}()

	defer func() {
		for retry := 0; retry < 3; retry++ {
			msg := NewMessage(l.MessID, MessageTypeRemoveConnect, nil)
			if err := msg.WriteTo(tunnel); err != nil {
				klog.Errorf("%v send %s message error %v", l, msg.MessageType, err)
			} else {
				break
			}
		}
	}()

	for {
		select {
		case <-stop:
			klog.Infof("receive stop single, so stop logs scan ...")
			return nil
		default:
		}
		data := make([]byte, 256)

		n, err := reader.Read(data)
		if err != nil {
			if err != io.EOF {
				klog.Errorf("%v failed to write log data, err:%v", l.String(), err)
			}
			break
		}
		if n <= 0 {
			continue
		}
		msg := NewMessage(l.MessID, MessageTypeData, data[:n])

		err = msg.WriteTo(tunnel)
		if err != nil {
			klog.Errorf("write tunnel message %v error", msg)
			return err
		}
		klog.V(4).Infof("%v write logs %v", l.String(), string(data))
	}
	return nil
}

var _ EdgedConnection = &EdgedLogsConnection{}
