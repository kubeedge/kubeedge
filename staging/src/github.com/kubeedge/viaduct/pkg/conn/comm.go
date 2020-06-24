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

package conn

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/lane"
)

type responseWriter struct {
	Type string
	Van  interface{}
}

// write response
func (r *responseWriter) WriteResponse(msg *model.Message, content interface{}) {
	response := msg.NewRespByMessage(msg, content)
	err := lane.NewLane(r.Type, r.Van).WriteMessage(response)
	if err != nil {
		klog.Errorf("failed to write response, error: %+v", err)
	}
}

// write error
func (r *responseWriter) WriteError(msg *model.Message, errMsg string) {
	response := model.NewErrorMessage(msg, errMsg)
	err := lane.NewLane(r.Type, r.Van).WriteMessage(response)
	if err != nil {
		klog.Errorf("failed to write error, error: %+v", err)
	}
}
