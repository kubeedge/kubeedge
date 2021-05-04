package config

import (
	"strings"
	"sync"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgeHub
	WebSocketURL string
	NodeName     string
}

func InitConfigure(eh *v1alpha1.EdgeHub, nodeName string) {
	once.Do(func() {
		Config = Configure{
			EdgeHub:      *eh,
			WebSocketURL: strings.Join([]string{"wss:/", eh.WebSocket.Server, eh.ProjectID, nodeName, "events"}, "/"),
			NodeName:     nodeName,
		}
	})
}
