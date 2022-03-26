/*
Copyright 2020 The Kubernetes Authors.

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

package util

import (
	"crypto/tls"
	"strings"
)

// containIPv6Addr checks if the given host identity contains an
// IPv6 address
func containIPv6Addr(host string) bool {
	// the shortest IPv6 address is ::
	return len(strings.Split(host, ":")) > 2
}

// containPortIPv6 checks if the host that contains an IPv6 address
// also contains a port number
func containPortIPv6(host string) bool {
	// based on to RFC 3986, section 3.2.2, host identified by an
	// IPv6 is distinguished by enclosing the IP literal within square
	// brackets ("[" and "]")
	return strings.ContainsRune(host, '[')
}

// RemovePortFromHost removes port number from the host address that
// may be of the form "<host>:<port>" where the <host> can be an either
// an IPv4/6 address or a domain name
func RemovePortFromHost(host string) string {
	if !containIPv6Addr(host) {
		return strings.Split(host, ":")[0]
	}
	if containPortIPv6(host) {
		host = host[:strings.LastIndexByte(host, ':')]
	}
	return strings.Trim(host, "[]")
}

// GetAcceptedCiphers returns all the ciphers supported by the crypto/tls package
func GetAcceptedCiphers() map[string]uint16 {
	acceptedCiphers := make(map[string]uint16, len(tls.CipherSuites()))
	for _, v := range tls.CipherSuites() {
		acceptedCiphers[v.Name] = v.ID
	}
	return acceptedCiphers
}
