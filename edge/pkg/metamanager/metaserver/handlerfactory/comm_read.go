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
