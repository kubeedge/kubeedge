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
	"fmt"
	"io"
	"net/http"

	"k8s.io/klog/v2"
)

type EdgedLogsConnection struct {
	BaseEdgedConnection `json:",inline"`
}

func (el *EdgedLogsConnection) CreateConnectMessage() (*Message, error) {
	return el.createConnectMessage(MessageTypeLogsConnect, el)
}

func (el *EdgedLogsConnection) String() string {
	return fmt.Sprintf("EDGE_LOGS_CONNECTOR Message MessageID %v", el.MessID)
}

func (el *EdgedLogsConnection) receiveFromCloudStream() {
	for mess := range el.ReadChan {
		if mess.MessageType == MessageTypeRemoveConnect {
			klog.Infof("receive remove client id %v", mess.ConnectID)
			el.Stop <- struct{}{}
		}
	}
	klog.V(2).Infof("%s read channel closed", el.String())
}

func (el *EdgedLogsConnection) write2CloudStream(tunnel SafeWriteTunneler, resp *http.Response) {
	defer func() {
		el.Stop <- struct{}{}
	}()
	data := make([]byte, 256)
	reader := bufio.NewReader(resp.Body)
	for {
		n, err := reader.Read(data)
		if err != nil {
			if err == io.EOF {
				klog.V(2).Info("trigger EOF when read response body")
				if n > 0 {
					el.writeMessage(tunnel, data[:n])
				}
				return
			}
			klog.Errorf("%s failed to read log data, err: %v", el.String(), err)
			return
		}
		if n < 1 {
			klog.V(2).Infof("%s read zero value, break the loop", el.String())
			return
		}
		el.writeMessage(tunnel, data[:n])
	}
}

func (el *EdgedLogsConnection) writeMessage(tunnel SafeWriteTunneler, data []byte) {
	msg := NewMessage(el.MessID, MessageTypeData, data)
	if err := tunnel.WriteMessage(msg); err != nil {
		klog.Errorf("write tunnel message %v error", msg)
		return
	}
	klog.V(4).Infof("%s write logs %s", el.String(), data)
}

func (el *EdgedLogsConnection) Serve(tunnel SafeWriteTunneler) error {
	return el.serveByClient(tunnel, httpClientCustomization{
		name:                   el.String(),
		receiveFromCloudStream: el.receiveFromCloudStream,
		write2CloudStream:      el.write2CloudStream,
	})
}

var _ EdgedConnection = &EdgedLogsConnection{}
