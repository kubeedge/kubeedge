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

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// quic client option
// extend options when you using quic in client
type QuicClientOption struct {
	// send heads after connection completed using control stream
	// TODO:
	Header http.Header
	// the max incoming stream
	MaxIncomingStreams int
}

// you can do some additional processes after successful dialing
type WSClientCallback func(*websocket.Conn, *http.Response)

// websocket client options
// extend options when you using websocket in client
type WSClientOption struct {
	// extend headers that you want to input
	Header http.Header
	// called after dialing
	Callback WSClientCallback
}
