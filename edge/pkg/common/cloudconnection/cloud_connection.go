/*
Copyright 2022 The KubeEdge Authors.

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
