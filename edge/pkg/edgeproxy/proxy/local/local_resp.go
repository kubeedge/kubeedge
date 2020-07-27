package local

import (
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
)

func (p *Proxy) WriteObject(statusCode int, obj runtime.Object, w http.ResponseWriter, req *http.Request) {
	gv := p.getRequestGroupVersion(req)
	responsewriters.WriteObjectNegotiated(clientscheme.Codecs, negotiation.DefaultEndpointRestrictions, gv, w, req, statusCode, obj)
}

func (p *Proxy) Err(err error, w http.ResponseWriter, req *http.Request) {
	gv := p.getRequestGroupVersion(req)
	responsewriters.ErrorNegotiated(err, clientscheme.Codecs, gv, w, req)
}

func (p *Proxy) getRequestGroupVersion(req *http.Request) schema.GroupVersion {
	ctx := req.Context()
	gv := schema.GroupVersion{
		Group:   "",
		Version: runtime.APIVersionInternal,
	}
	if info, ok := apirequest.RequestInfoFrom(ctx); ok {
		gv.Group = info.APIGroup
		gv.Version = info.APIVersion
	}
	return gv
}
