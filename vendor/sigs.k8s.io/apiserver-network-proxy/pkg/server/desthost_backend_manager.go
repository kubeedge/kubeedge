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

package server

import (
	"context"

	"k8s.io/klog/v2"
	"sigs.k8s.io/apiserver-network-proxy/pkg/agent"
)

type DestHostBackendManager struct {
	*DefaultBackendStorage
}

var _ BackendManager = &DestHostBackendManager{}

func NewDestHostBackendManager() *DestHostBackendManager {
	return &DestHostBackendManager{
		DefaultBackendStorage: NewDefaultBackendStorage(
			[]agent.IdentifierType{agent.IPv4, agent.IPv6, agent.Host})}
}

// Backend tries to get a backend associating to the request destination host.
func (dibm *DestHostBackendManager) Backend(ctx context.Context) (Backend, error) {
	dibm.mu.RLock()
	defer dibm.mu.RUnlock()
	if len(dibm.backends) == 0 {
		return nil, &ErrNotFound{}
	}
	destHost := ctx.Value(destHost).(string)
	if destHost != "" {
		bes, exist := dibm.backends[destHost]
		if exist && len(bes) > 0 {
			klog.V(5).InfoS("Get the backend through the DestHostBackendManager", "destHost", destHost)
			return dibm.backends[destHost][0], nil
		}
	}
	return nil, &ErrNotFound{}
}
