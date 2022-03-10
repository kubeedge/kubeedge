package cloudconnection

import "sync"

// constants for cloud connection
const (
	CloudConnected    = "cloud_connected"
	CloudDisconnected = "cloud_disconnected"
)

var (
	// isCloudConnected indicate Whether the connection was
	// successfully established between edge and cloud
	isCloudConnected = false

	lock sync.RWMutex
)

// SetConnected set isCloudConnected value
// true indicates edge and cloud establish connection successfully
// false indicates edge and cloud connection interrupted.
func SetConnected(isConnected bool) {
	lock.Lock()
	defer lock.Unlock()
	isCloudConnected = isConnected
}

// IsConnected return isCloudConnected
func IsConnected() bool {
	lock.RLock()
	defer lock.RUnlock()
	return isCloudConnected
}
