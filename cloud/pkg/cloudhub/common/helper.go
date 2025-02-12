/*
Copyright 2022 The KubeEdge Authors.

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

package common

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	edgecon "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn"
)

// VolumePattern constants for error message
const (
	VolumePattern = `^\w[-\w.+]*/` + constants.CSIResourceTypeVolume + `/\w[-\w.+]*`
)

// VolumeRegExp is used to validate the volume resource
var VolumeRegExp = regexp.MustCompile(VolumePattern)

func IsVolumeResource(resource string) bool {
	return VolumeRegExp.MatchString(resource)
}

// GetMessageUID returns the UID of the object in message
func GetMessageUID(msg beehivemodel.Message) (string, error) {
	accessor, err := meta.Accessor(msg.Content)
	if err != nil {
		return "", err
	}

	return string(accessor.GetUID()), nil
}

// GetMessageDeletionTimestamp returns the deletionTimestamp of the object in message
func GetMessageDeletionTimestamp(msg *beehivemodel.Message) (*v1.Time, error) {
	accessor, err := meta.Accessor(msg.Content)
	if err != nil {
		return nil, err
	}

	return accessor.GetDeletionTimestamp(), nil
}

// TrimMessage trims resource field in message
// before: node/{nodename}/{namespace}/pod/{podname}
// after: {namespace}/pod/{podname}
func TrimMessage(msg *beehivemodel.Message) {
	resource := msg.GetResource()
	if strings.HasPrefix(resource, model.ResNode) {
		tokens := strings.Split(resource, "/")
		if len(tokens) < 3 {
			klog.Warningf("event resource %s starts with node but length less than 3", resource)
		} else {
			msg.SetResourceOperation(strings.Join(tokens[2:], "/"), msg.GetOperation())
		}
	}
}

func ConstructConnectMessage(info *model.HubInfo, isConnected bool) *beehivemodel.Message {
	connected := model.OpConnect
	if !isConnected {
		connected = model.OpDisConnect
	}
	body := map[string]interface{}{
		"event_type": connected,
		"timestamp":  time.Now().Unix(),
		"client_id":  info.NodeID,
	}
	content, _ := json.Marshal(body)

	msg := beehivemodel.NewMessage("")
	msg.BuildRouter(model.SrcCloudHub, model.GpResource, model.NewResource(model.ResNode, info.NodeID, nil), connected)
	msg.FillBody(content)
	return msg
}

func DeepCopy(msg *beehivemodel.Message) *beehivemodel.Message {
	if msg == nil {
		return nil
	}
	out := new(beehivemodel.Message)
	out.Header = msg.Header
	out.Router = msg.Router
	out.Content = msg.Content
	return out
}

func NotifyEventQueueError(conn conn.Connection, nodeID string) {
	msg := beehivemodel.NewMessage("").BuildRouter(model.GpResource, model.SrcCloudHub,
		model.NewResource(model.ResNode, nodeID, nil), model.OpDisConnect)

	err := conn.WriteMessageAsync(msg)
	if err != nil {
		klog.Errorf("fail to notify node %s event queue disconnected, reason: %s", nodeID, err.Error())
	}
}

func AckMessageKeyFunc(obj interface{}) (string, error) {
	msg, ok := obj.(*beehivemodel.Message)
	if !ok {
		return "", fmt.Errorf("object type %T is not message type", msg)
	}

	if msg.GetGroup() == edgecon.GroupResource {
		return GetMessageUID(*msg)
	}

	return "", fmt.Errorf("failed to get message key")
}

func NoAckMessageKeyFunc(obj interface{}) (string, error) {
	msg, ok := obj.(*beehivemodel.Message)
	if !ok {
		return "", fmt.Errorf("object is not message type")
	}

	return msg.Header.ID, nil
}
