/*
Copyright 2025 The KubeEdge Authors.

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

package handlerfactory

import (
	"net/http"

	"k8s.io/apiserver/pkg/endpoints/handlers"
)

func (f *Factory) Get() http.Handler {
	if h, ok := f.getHandler("get"); ok {
		return h
	}
	f.lock.Lock()
	defer f.lock.Unlock()
	h := handlers.GetResource(f.storage, f.scope)
	f.handlers["get"] = h
	return h
}

func (f *Factory) List() http.Handler {
	if h, ok := f.getHandler("list"); ok {
		return h
	}
	f.lock.Lock()
	defer f.lock.Unlock()
	h := handlers.ListResource(f.storage, f.storage, f.scope, false, f.MinRequestTimeout)
	f.handlers["list"] = h
	return h
}
