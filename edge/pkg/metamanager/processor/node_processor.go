package processor

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	metaManagerConfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/config"
)

// nodeConnectionProcessor process node connection
type nodeConnectionProcessor struct {
}

func (m *nodeConnectionProcessor) Process(message model.Message) {
	content, _ := message.GetContent().(string)
	klog.Infof("node connection event occur: %s", content)
	if content == connect.CloudConnected {
		metaManagerConfig.Connected = true
	} else if content == connect.CloudDisconnected {
		metaManagerConfig.Connected = false
	}
}

func init() {
	ncKey := queryKey{
		operation: messagepkg.OperationNodeConnection,
	}

	processors[ncKey] = &nodeConnectionProcessor{}
}
