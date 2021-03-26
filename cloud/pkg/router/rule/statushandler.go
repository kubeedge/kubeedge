package rule

import (
	"time"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/messagelayer"
)

type ExecResult struct {
	RuleID    string
	ProjectID string
	Status    string
	Error     ErrorMsg
}

type ErrorMsg struct {
	Detail    string
	Timestamp time.Time
}

var ResultChannel chan ExecResult
var StopChan chan bool

func init() {
	StopChan = make(chan bool)
	go do(StopChan)
}

func do(stop chan bool) {
	ResultChannel = make(chan ExecResult, 1024)
	for {
		select {
		case r := <-ResultChannel:
			msg := model.NewMessage("")
			resource, err := messagelayer.BuildResourceForRouter(r.ProjectID, model.ResourceTypeRuleStatus, r.RuleID)
			if err != nil {
				klog.Warningf("build message resource failed with error: %s", err)
				continue
			}
			msg.Content = r
			msg.BuildRouter(modules.RouterModuleName, constants.GroupResource, resource, model.UpdateOperation)
			beehiveContext.Send(modules.EdgeControllerModuleName, *msg)
			klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
		case _, ok := <-stop:
			if !ok {
				klog.Warningf("do stop channel is closed")
			}
		}
	}
}
