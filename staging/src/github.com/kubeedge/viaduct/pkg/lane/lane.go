/*
Copyright 2019 The KubeEdge Authors.

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

package lane

import (
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/api"
)

type Lane interface {
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	ReadMessage(msg *model.Message) error
	WriteMessage(msg *model.Message) error
	Read(raw []byte) (int, error)
	Write(raw []byte) (int, error)
}

func NewLane(protoType string, van interface{}) Lane {
	switch protoType {
	case api.ProtocolTypeQuic:
		return NewQuicLane(van)
	case api.ProtocolTypeWS:
		return NewWSLaneWithoutPack(van)
	}
	klog.Errorf("bad protocol type(%s)", protoType)
	return nil
}
