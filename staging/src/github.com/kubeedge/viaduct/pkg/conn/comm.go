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
