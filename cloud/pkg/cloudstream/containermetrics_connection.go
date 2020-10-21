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

package cloudstream

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/emicklei/go-restful"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

// ContainerMetricsConnection indicates the containerMetrics request initiated by kube-apiserver
type ContainerMetricsConnection struct {
	// MessageID indicate the unique id to create his message
	MessageID    uint64
	ctx          context.Context
	r            *restful.Request
	writer       io.Writer
	session      *Session
	edgePeerStop chan struct{}
}

func (ms *ContainerMetricsConnection) GetMessageID() uint64 {
	return ms.MessageID
}

func (ms *ContainerMetricsConnection) SetEdgePeerDone() {
	close(ms.edgePeerStop)
}

func (ms *ContainerMetricsConnection) EdgePeerDone() <-chan struct{} {
	return ms.edgePeerStop
}

func (ms *ContainerMetricsConnection) WriteToAPIServer(p []byte) (n int, err error) {
	return ms.writer.Write(p)
}

func (ms *ContainerMetricsConnection) WriteToTunnel(m *stream.Message) error {
	return ms.session.WriteMessageToTunnel(m)
}

func (ms *ContainerMetricsConnection) SetMessageID(id uint64) {
	ms.MessageID = id
}

func (ms *ContainerMetricsConnection) String() string {
	return fmt.Sprintf("APIServer_MetricsConnection MessageID %v", ms.MessageID)
}

func (ms *ContainerMetricsConnection) SendConnection() (stream.EdgedConnection, error) {
	connector := &stream.EdgedMetricsConnection{
		MessID: ms.MessageID,
		URL:    *ms.r.Request.URL,
		Header: ms.r.Request.Header,
	}
	targetPort := strings.Split(ms.r.Request.Host, ":")[1]
	connector.URL.Scheme = httpScheme
	connector.URL.Host = net.JoinHostPort(defaultServerHost, targetPort)
	m, err := connector.CreateConnectMessage()
	if err != nil {
		return nil, err
	}
	if err := ms.WriteToTunnel(m); err != nil {
		klog.Errorf("%s write %s error %v", ms.String(), connector.String(), err)
		return nil, err
	}
	return connector, nil
}

func (ms *ContainerMetricsConnection) Serve() error {
	defer func() {
		klog.Infof("%s end successful", ms.String())
	}()

	// first send connect message
	if _, err := ms.SendConnection(); err != nil {
		klog.Errorf("%s send %s info error %v", ms.String(), stream.MessageTypeMetricConnect, err)
		return err
	}

	for {
		select {
		case <-ms.ctx.Done():
			// if apiserver request end, send close message to edge
			msg := stream.NewMessage(ms.MessageID, stream.MessageTypeRemoveConnect, nil)
			for retry := 0; retry < 3; retry++ {
				if err := ms.WriteToTunnel(msg); err != nil {
					klog.Warningf("%v send %s message to edge error %v", ms, msg.MessageType, err)
				} else {
					break
				}
			}
			klog.Infof("%s send close message to edge successfully", ms.String())
			return nil
		case <-ms.EdgePeerDone():
			klog.Infof("%s find edge peer done, so stop this connection", ms.String())
			return nil
		}
	}
}

var _ APIServerConnection = &ContainerMetricsConnection{}
