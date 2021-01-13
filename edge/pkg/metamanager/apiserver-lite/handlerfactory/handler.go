package handlerfactory

import (
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/apiserver-lite/kubernetes/scope"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/apiserver-lite/kubernetes/storage"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"net/http"
	"time"
)

type Factory struct{
	storage  *storage.REST
	scope *handlers.RequestScope
	MinRequestTimeout time.Duration
}
func NewFactory() Factory {
	s,err := storage.NewREST()
	utilruntime.Must(err)
	f := Factory{
		storage:           s,
		scope:             scope.NewRequestScope(),
		MinRequestTimeout: 1800*time.Second,
	}
	return f
}
func (f *Factory)List()http.Handler{
	h := handlers.ListResource(f.storage,f.storage,f.scope, false,f.MinRequestTimeout)
	return h
}
func (f *Factory)Get()http.Handler{
	h := handlers.GetResource(f.storage,f.storage,f.scope)
	return h
}
