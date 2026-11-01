/*
Copyright 2024 The KubeEdge Authors.

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
package certs

const (
	// Untyped string constants preserve source compatibility for downstream users
	// who pass these to functions accepting string or named string types.
	CAHandlerTypeX509 = "x509"

	HandlerTypeX509 = "x509"
)

// CAHandlerType is the type for CA certificate handler identifiers.
type CAHandlerType string

// HandlerType is the corrected type for certificate handler identifiers.
type HandlerType string

// Deprecated: HanndlerType is a typo alias for HandlerType kept for backward compatibility.
// Use HandlerType instead.
type HanndlerType = HandlerType

func GetCAHandler(t CAHandlerType) CAHandler {
	switch t {
	case CAHandlerTypeX509:
		return &x509CAHandler{}
	}
	return nil
}

func GetHandler(t HandlerType) Handler {
	switch t {
	case HandlerTypeX509:
		return &x509CertsHandler{}
	}
	return nil
}
