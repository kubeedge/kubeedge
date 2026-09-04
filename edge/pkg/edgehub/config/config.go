package config

import (
	"strings"
	"sync"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
)

var Config Configure
var once sync.Once

// heartbeatMu guards Heartbeat since it can be updated at runtime by a
// configuration hot reload while the heartbeat loop in process.go reads it
// concurrently.
var heartbeatMu sync.RWMutex

type Configure struct {
	v1alpha2.EdgeHub
	WebSocketURL string
	NodeName     string
}

func InitConfigure(eh *v1alpha2.EdgeHub, nodeName string) {
	once.Do(func() {
		Config = Configure{
			EdgeHub:      *eh,
			WebSocketURL: strings.Join([]string{"wss:/", eh.WebSocket.Server, eh.ProjectID, nodeName, "events"}, "/"),
			NodeName:     nodeName,
		}
	})
}

// GetHeartbeat returns the current heartbeat interval in seconds.
func GetHeartbeat() int32 {
	heartbeatMu.RLock()
	defer heartbeatMu.RUnlock()
	return Config.Heartbeat
}

// SetHeartbeat updates the heartbeat interval in seconds. It is safe to call
// while the edgehub module is running, allowing the value to be hot reloaded
// without restarting edgecore.
func SetHeartbeat(heartbeat int32) {
	heartbeatMu.Lock()
	defer heartbeatMu.Unlock()
	Config.Heartbeat = heartbeat
}
