/*
Copyright 2026 The KubeEdge Authors.

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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
)

// recordingMessageLayer records the responses it is asked to send.
type recordingMessageLayer struct {
	responses []model.Message
}

func (r *recordingMessageLayer) Send(model.Message) error { return nil }

func (r *recordingMessageLayer) Receive() (model.Message, error) { return model.Message{}, nil }

func (r *recordingMessageLayer) Response(message model.Message) error {
	r.responses = append(r.responses, message)
	return nil
}

func TestAckMembershipDetail(t *testing.T) {
	ml := &recordingMessageLayer{}
	uc := &UpstreamController{messageLayer: ml}

	req := model.NewMessage("")
	req.BuildRouter("edgehub", constants.GroupTwin, "node/edge-1/membership/detail", model.QueryOperation)
	uc.ackMembershipDetail(*req)

	if assert.Len(t, ml.responses, 1) {
		resp := ml.responses[0]
		// edge devicetwin drops the request from its confirm map only on this parent ID
		assert.Equal(t, req.GetID(), resp.GetParentID())
		assert.Equal(t, constants.GroupTwin, resp.GetGroup())
		assert.Equal(t, modules.DeviceControllerModuleName, resp.GetSource())
		assert.Equal(t, "node/edge-1/membership/detail", resp.GetResource())
		assert.Equal(t, model.ResponseOperation, resp.GetOperation())
		assert.Equal(t, commonconst.MessageSuccessfulContent, resp.GetContent())
	}
}

func TestAckMembershipDetailWithoutNodeID(t *testing.T) {
	ml := &recordingMessageLayer{}
	uc := &UpstreamController{messageLayer: ml}

	req := model.NewMessage("")
	req.BuildRouter("edgehub", constants.GroupTwin, "", model.QueryOperation)
	uc.ackMembershipDetail(*req)

	assert.Empty(t, ml.responses)
}
