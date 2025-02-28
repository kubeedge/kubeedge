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
	"errors"
	"fmt"
	"io"
	"net"

	"k8s.io/klog/v2"
)

type EdgedAttachConnection struct {
	BaseEdgedConnection `json:",inline"`
}

func (ah *EdgedAttachConnection) CreateConnectMessage() (*Message, error) {
	return ah.createConnectMessage(MessageTypeAttachConnect, ah)
}

func (ah *EdgedAttachConnection) String() string {
	return fmt.Sprintf("EDGE_ATTACH_CONNECTOR Message MessageID %v", ah.MessID)
}

func (ah *EdgedAttachConnection) receiveFromCloudStream(con net.Conn) {
	for message := range ah.ReadChan {
		switch message.MessageType {
		case MessageTypeRemoveConnect:
			klog.V(6).Infof("%s receive remove client id %v", ah.String(), message.ConnectID)
			ah.Stop <- struct{}{}
		case MessageTypeData:
			_, err := con.Write(message.Data)
			klog.V(6).Infof("%s receive attach %v data ", ah.String(), message.Data)
			if err != nil {
				klog.Errorf("failed to write, err: %v", err)
			}
		}
	}
	klog.V(2).Infof("%s read channel closed", ah.String())
}

func (ah *EdgedAttachConnection) write2CloudStream(tunnel SafeWriteTunneler, con net.Conn) {
	defer func() {
		ah.Stop <- struct{}{}
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
	return ah.serveByRoundTripper(tunnel, roundTripperCustomization{
		name:                   ah.String(),
		receiveFromCloudStream: ah.receiveFromCloudStream,
		write2CloudStream:      ah.write2CloudStream,
	})
}

var _ EdgedConnection = &EdgedAttachConnection{}
