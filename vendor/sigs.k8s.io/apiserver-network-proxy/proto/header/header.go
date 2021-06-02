/*
Copyright 2019 The Kubernetes Authors.

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

package header

const (
	ServerCount      = "serverCount"
	ServerID         = "serverID"
	AgentID          = "agentID"
	AgentIdentifiers = "agentIdentifiers"
	// AuthenticationTokenContextKey will be used as a key to store authentication tokens in grpc call
	// (https://tools.ietf.org/html/rfc6750#section-2.1)
	AuthenticationTokenContextKey = "Authorization"

	// AuthenticationTokenContextSchemePrefix has a prefix for auth token's content.
	// (https://tools.ietf.org/html/rfc6750#section-2.1)
	AuthenticationTokenContextSchemePrefix = "Bearer "

	// UserAgent is used to provide the client information in a proxy request
	UserAgent = "user-agent"
)
