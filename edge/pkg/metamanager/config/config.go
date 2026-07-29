package config

import (
	"sync"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
)

var Config Configure
var once sync.Once

// remoteQueryTimeoutMu guards RemoteQueryTimeout since it can be updated at
// runtime by a configuration hot reload while process.go reads it
// concurrently for each remote query.
var remoteQueryTimeoutMu sync.RWMutex

type Configure struct {
	v1alpha2.MetaManager
}

func InitConfigure(m *v1alpha2.MetaManager) {
	once.Do(func() {
		Config = Configure{
			MetaManager: *m,
		}
		metaserverconfig.InitConfigure(Config.MetaManager.MetaServer)
	})
}

// GetRemoteQueryTimeout returns the current remote query timeout in seconds.
func GetRemoteQueryTimeout() int32 {
	remoteQueryTimeoutMu.RLock()
	defer remoteQueryTimeoutMu.RUnlock()
	return Config.RemoteQueryTimeout
}

// SetRemoteQueryTimeout updates the remote query timeout in seconds. It is
// safe to call while the metamanager module is running, allowing the value
// to be hot reloaded without restarting edgecore.
func SetRemoteQueryTimeout(timeout int32) {
	remoteQueryTimeoutMu.Lock()
	defer remoteQueryTimeoutMu.Unlock()
	Config.RemoteQueryTimeout = timeout
}
