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
	"regexp"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	hubmodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/common/constants"
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
func GetMessageUID(msg model.Message) (string, error) {
	accessor, err := meta.Accessor(msg.Content)
	if err != nil {
		return "", err
	}

	return string(accessor.GetUID()), nil
}

// GetMessageDeletionTimestamp returns the deletionTimestamp of the object in message
func GetMessageDeletionTimestamp(msg *model.Message) (*v1.Time, error) {
	accessor, err := meta.Accessor(msg.Content)
	if err != nil {
		return nil, err
	}

	return accessor.GetDeletionTimestamp(), nil
}

func TrimMessage(msg *model.Message) {
	resource := msg.GetResource()
	if strings.HasPrefix(resource, hubmodel.ResNode) {
		tokens := strings.Split(resource, "/")
		if len(tokens) < 3 {
			klog.Warningf("event resource %s starts with node but length less than 3", resource)
		} else {
			msg.SetResourceOperation(strings.Join(tokens[2:], "/"), msg.GetOperation())
		}
	}
}

func ConstructConnectMessage(info *hubmodel.HubInfo, isConnected bool) *model.Message {
	connected := hubmodel.OpConnect
	if !isConnected {
		connected = hubmodel.OpDisConnect
	}
	body := map[string]interface{}{
		"event_type": connected,
		"timestamp":  time.Now().Unix(),
		"client_id":  info.NodeID}
	content, _ := json.Marshal(body)

	msg := model.NewMessage("")
	msg.BuildRouter(hubmodel.SrcCloudHub, hubmodel.GpResource, hubmodel.NewResource(hubmodel.ResNode, info.NodeID, nil), connected)
	msg.FillBody(content)
	return msg
}
