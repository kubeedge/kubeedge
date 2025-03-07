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
	"net/http"
	"net/url"

	"k8s.io/klog/v2"
)

type EdgedMetricsConnection struct {
	ReadChan chan *Message `json:"-"`
	Stop     chan struct{} `json:"-"`
	MessID   uint64        // message id
	URL      url.URL       `json:"url"`
	Header   http.Header   `json:"header"`
}

func (ms *EdgedMetricsConnection) GetMessageID() uint64 {
	return ms.MessID
}

func (ms *EdgedMetricsConnection) CacheTunnelMessage(msg *Message) {
	ms.ReadChan <- msg
}

func (ms *EdgedMetricsConnection) CloseReadChannel() {
	close(ms.ReadChan)
}

func (ms *EdgedMetricsConnection) CleanChannel() {
	for {
		select {
		case <-ms.Stop:
		default:
			return
		}
	}
}

func (ms *EdgedMetricsConnection) CreateConnectMessage() (*Message, error) {
	data, err := json.Marshal(ms)
	if err != nil {
		return nil, err
	}
	return NewMessage(ms.MessID, MessageTypeMetricConnect, data), nil
}

func (ms *EdgedMetricsConnection) String() string {
	return fmt.Sprintf("EDGE_METRICS_CONNECTOR Message MessageID %v", ms.MessID)
}

func (ms *EdgedMetricsConnection) receiveFromCloudStream(stop chan struct{}) {
	for mess := range ms.ReadChan {
		if mess.MessageType == MessageTypeRemoveConnect {
			klog.Infof("receive remove client id %v", mess.ConnectID)
			stop <- struct{}{}
		}
	}
	klog.V(6).Infof("%s read channel closed", ms.String())
}

func (ms *EdgedMetricsConnection) write2CloudStream(tunnel SafeWriteTunneler, resp *http.Response, stop chan struct{}) {
	defer func() {
		stop <- struct{}{}
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
	//connect edged
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, ms.URL.String(), nil)
	if err != nil {
		klog.Errorf("create new metrics request error %v", err)
		return err
	}
	req.Header = ms.Header
	// Since current tunnel implementation only support Text message,
	// we should force Accept-Encoding to identity to avoid any compression.
	// For example, user may pass accept-encoding: gzip in header.
	// TODO: luogangyi
	// When we support binary message, we can remove this setting.
	req.Header.Set("accept-encoding", "identity")
	resp, err := client.Do(req)
	if err != nil {
		klog.Errorf("request metrics error %v", err)
		return err
	}
	defer resp.Body.Close()

	go ms.receiveFromCloudStream(ms.Stop)

	defer func() {
		for retry := 0; retry < 3; retry++ {
			msg := NewMessage(ms.MessID, MessageTypeRemoveConnect, nil)
			if err := tunnel.WriteMessage(msg); err != nil {
				klog.Errorf("%v send %s message error %v", ms, msg.MessageType, err)
			} else {
				break
			}
		}
	}()

	go ms.write2CloudStream(tunnel, resp, ms.Stop)

	<-ms.Stop
	klog.Infof("receive stop signal, so stop metrics scan ...")
	return nil
}

var _ EdgedConnection = &EdgedMetricsConnection{}
