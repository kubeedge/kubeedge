package streamrule

import (
	"time"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/messagelayer"
	"k8s.io/klog/v2"
)

type ExecResult struct {
	StreamruleID string
	Namespace    string
	Status       string
	Error        ErrorMsg
}

type ErrorMsg struct {
	Detail    string
	Timestamp time.Time
}

var ResultChannel chan ExecResult
var StopChan chan bool

func init() {
	StopChan = make(chan bool)
	go SendMessageToController(StopChan)
}

func SendMessageToController(stop chan bool) {
	ResultChannel = make(chan ExecResult, 1024)
	for {
		select {
		case r := <-ResultChannel:
			msg := model.NewMessage("")
			resource, err := messagelayer.BuildResourceForStreamRuleController(r.Namespace, "streamrulestatus", r.StreamruleID)
			if err != nil {
				klog.Warningf("build message resource failed with error: %s", err)
				continue
			}
			msg.Content = r
			msg.BuildRouter(modules.StreamRuleControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
			beehiveContext.Send(modules.EdgeControllerModuleName, *msg)
			klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())

		case _, ok := <-stop:
			if !ok {
				klog.Warningf("do stop channel is closed")
			}
		}
	}
}
