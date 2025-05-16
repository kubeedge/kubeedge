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
	"net/http"

	"k8s.io/klog/v2"
)

type EdgedMetricsConnection struct {
	BaseEdgedConnection `json:",inline"`
}

func (ms *EdgedMetricsConnection) CreateConnectMessage() (*Message, error) {
	return ms.createConnectMessage(MessageTypeMetricConnect, ms)
}

func (ms *EdgedMetricsConnection) String() string {
	return fmt.Sprintf("EDGE_METRICS_CONNECTOR Message MessageID %v", ms.MessID)
}

func (ms *EdgedMetricsConnection) receiveFromCloudStream() {
	for mess := range ms.ReadChan {
		if mess.MessageType == MessageTypeRemoveConnect {
			klog.Infof("receive remove client id %v", mess.ConnectID)
			ms.Stop <- struct{}{}
		}
	}
	klog.V(2).Infof("%s read channel closed", ms.String())
}

func (ms *EdgedMetricsConnection) write2CloudStream(tunnel SafeWriteTunneler, resp *http.Response) {
	defer func() {
		ms.Stop <- struct{}{}
	}()
	scan := bufio.NewScanner(resp.Body)
	for scan.Scan() {
		// 10 = \n
		msg := NewMessage(ms.MessID, MessageTypeData, append(scan.Bytes(), 10))
		err := tunnel.WriteMessage(msg)
		if err != nil {
			klog.Errorf("write tunnel message %v error", msg)
			return
		}
		klog.V(4).Infof("%v write metrics data %v", ms.String(), string(scan.Bytes()))
	}
}

func (ms *EdgedMetricsConnection) Serve(tunnel SafeWriteTunneler) error {
	return ms.serveByClient(tunnel, httpClientCustomization{
		name: ms.String(),
		handleRequest: func(r *http.Request) {
			// Since current tunnel implementation only support Text message,
			// we should force Accept-Encoding to identity to avoid any compression.
			// For example, user may pass accept-encoding: gzip in header.
			// TODO: When we support binary message, we can remove this setting.
			r.Header.Set("accept-encoding", "identity")
		},
		receiveFromCloudStream: ms.receiveFromCloudStream,
		write2CloudStream:      ms.write2CloudStream,
	})
}

var _ EdgedConnection = &EdgedMetricsConnection{}
