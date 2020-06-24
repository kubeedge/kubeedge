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

package api

import "net/http"

// quic server option
// including the extend options when getting server instance
// we can add the essential option into
type QuicServerOption struct {
	// the max incoming stream
	MaxIncomingStreams int
}

// the filter function before upgrading the http to websocket
type WSFilterFunc func(w http.ResponseWriter, r *http.Request) bool

// websocket server option
// you can add the extend options when getting websocket server instance
type WSServerOption struct {
	// the path that the client dialing
	Path string
	// the necessary processing before upgrading
	Filter WSFilterFunc
}
