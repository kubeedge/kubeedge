package metaserver

import (
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	utilwaitgroup "k8s.io/apimachinery/pkg/util/waitgroup"
	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/server"
	genericfilters "k8s.io/apiserver/pkg/server/filters"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/handlerfactory"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/serializer"
)

const (
	// TODO: make addrs configurable
	Httpaddr = "127.0.0.1:10550"
)

// MetaServer is simplification of server.GenericAPIServer
type MetaServer struct {
	HandlerChainWaitGroup *utilwaitgroup.SafeWaitGroup
	LongRunningFunc       apirequest.LongRunningRequestCheck
	RequestTimeout        time.Duration
	Handler               http.Handler
	NegotiatedSerializer  runtime.NegotiatedSerializer
	Factory               *handlerfactory.Factory
}

func NewMetaServer() *MetaServer {
	ls := MetaServer{
		HandlerChainWaitGroup: new(utilwaitgroup.SafeWaitGroup),
		LongRunningFunc:       genericfilters.BasicLongRunningRequestCheck(sets.NewString("watch"), sets.NewString()),
		NegotiatedSerializer:  serializer.NewNegotiatedSerializer(),
		Factory:               handlerfactory.NewFactory(),
	}
	return &ls
}

func (ls *MetaServer) Start(stopChan <-chan struct{}) {
	h := ls.BuildBasicHandler()
	h = BuildHandlerChain(h, ls)
	s := http.Server{
		Addr:    Httpaddr,
		Handler: h,
	}
	utilruntime.HandleError(s.ListenAndServe())
	klog.Infof("[metaserver]start to listen and server at %v", Httpaddr)
	<-stopChan
}

func (ls *MetaServer) BuildBasicHandler() http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		reqInfo, ok := apirequest.RequestInfoFrom(ctx)
		//klog.Infof("[metaserver]get a req(%v)(%v)", reqInfo.Path, reqInfo.Verb)
		//klog.Infof("[metaserver]get a req(\nPath:%v; \nVerb:%v; \nHeader:%+v)", reqInfo.Path, reqInfo.Verb, req.Header)
		if ok && reqInfo.IsResourceRequest {
			switch {
			case reqInfo.Verb == "get":
				ls.Factory.Get().ServeHTTP(w, req)
			case reqInfo.Verb == "list", reqInfo.Verb == "watch":
				ls.Factory.List().ServeHTTP(w, req)
			case reqInfo.Verb == "create":
				ls.Factory.Create(reqInfo).ServeHTTP(w, req)
			case reqInfo.Verb == "delete":
				ls.Factory.Delete().ServeHTTP(w, req)
			case reqInfo.Verb == "update":
				ls.Factory.Update(reqInfo).ServeHTTP(w, req)
			case reqInfo.Verb == "patch":
				ls.Factory.Patch(reqInfo).ServeHTTP(w, req)
			default:
				err := fmt.Errorf("unsupported req verb")
				responsewriters.ErrorNegotiated(errors.NewInternalError(err), ls.NegotiatedSerializer, schema.GroupVersion{}, w, req)
				return
			}
		} else {
			err := fmt.Errorf("not a resource req")
			responsewriters.ErrorNegotiated(errors.NewInternalError(err), ls.NegotiatedSerializer, schema.GroupVersion{}, w, req)
		}
	})
	return h
}

func BuildHandlerChain(handler http.Handler, ls *MetaServer) http.Handler {
	cfg := &server.Config{
		LegacyAPIGroupPrefixes: sets.NewString(server.DefaultLegacyAPIPrefix),
	}
	handler = genericfilters.WithWaitGroup(handler, ls.LongRunningFunc, ls.HandlerChainWaitGroup)
	handler = genericapifilters.WithRequestInfo(handler, server.NewRequestInfoResolver(cfg))
	handler = genericfilters.WithPanicRecovery(handler)
	return handler
}
