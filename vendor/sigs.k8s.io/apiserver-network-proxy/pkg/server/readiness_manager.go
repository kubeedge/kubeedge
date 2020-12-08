/*
copyright 2020 the kubernetes authors.

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package server

// ReadinessManager supports checking if the proxy server is ready.
type ReadinessManager interface {
	// Ready returns if the proxy server is ready. If not, also return an
	// error message.
	Ready() (bool, string)
}

var _ ReadinessManager = &DefaultBackendManager{}

func (s *DefaultBackendManager) Ready() (bool, string) {
	if s.NumBackends() == 0 {
		return false, "no connection to any proxy agent"
	}
	return true, ""
}
