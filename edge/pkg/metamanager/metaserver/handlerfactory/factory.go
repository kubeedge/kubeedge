package handlerfactory

import (
	"net/http"
	"sync"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/endpoints/handlers"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/scope"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage"
)

type Factory struct {
	storage           *storage.REST
	scope             *handlers.RequestScope
	MinRequestTimeout time.Duration
	handlers          map[string]http.Handler
	lock              sync.RWMutex
}

func NewFactory() *Factory {
	s, err := storage.NewREST()
	utilruntime.Must(err)
	f := Factory{
		storage:           s,
		scope:             scope.NewRequestScope(),
		MinRequestTimeout: 1800 * time.Second,
		handlers:          make(map[string]http.Handler),
		lock:              sync.RWMutex{},
	}
	return &f
}

func (f *Factory) getHandler(key string) (http.Handler, bool) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	h, ok := f.handlers[key]
	return h, ok
}
