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
	"fmt"
	"net"
	"net/http"
	"net/url"
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
	return NewMessage(ah.MessID, MessageTypeExecConnect, data), nil
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
	// todo: read msg from cloudstream
}

func (ah *EdgedAttachConnection) write2CloudStream(tunnel SafeWriteTunneler, con net.Conn, stop chan struct{}) {
	// todo: write response to tunnel
}

func (ah *EdgedAttachConnection) Serve(tunnel SafeWriteTunneler) error {
	// todo: serve msg from clousstream, send req to edged and write response to tunnel
	return nil
}

var _ EdgedConnection = &EdgedAttachConnection{}
