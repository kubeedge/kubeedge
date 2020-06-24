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

package comm

const (
	// control message type
	ControlTypeHeader = "header"
	ControlTypeConfig = "config"
	ControlTypePing   = "ping"
	ControlTypePong   = "pong"

	// control message action
	ControlActionHeader = "/control/header"
	ControlActionConfig = "/control/config"
	ControlActionPing   = "/control/ping"
	ControlActionPong   = "/control/pong"

	// response type
	RespTypeAck  = "ack"
	RespTypeNack = "nack"

	// the max size of message fifo
	MessageFiFoSizeMax = 100
)
