package metaserver

import (
	"errors"
	"net/http"
	"time"

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
	httpaddr = "127.0.0.1:10010"
)

// MetaServer is simplification of server.GenericAPIServer
type MetaServer struct {
	HandlerChainWaitGroup *utilwaitgroup.SafeWaitGroup
	LongRunningFunc       apirequest.LongRunningRequestCheck
	RequestTimeout        time.Duration
	Handler               http.Handler
	NegotiatedSerializer  runtime.NegotiatedSerializer
	Factory               handlerfactory.Factory
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
		Addr:    httpaddr,
		Handler: h,
	}
	utilruntime.HandleError(s.ListenAndServe())
	<-stopChan
}

func (ls *MetaServer) BuildBasicHandler() http.Handler {
	listHandler := ls.Factory.List()
	getHandler := ls.Factory.Get()
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		reqInfo, ok := apirequest.RequestInfoFrom(ctx)
		klog.Infof("[metaserver]get a req(%v)(%v)", req.URL.RawPath, req.URL.RawQuery)
		if ok && reqInfo.IsResourceRequest {
			switch {
			case reqInfo.Verb == "get":
				getHandler.ServeHTTP(w, req)
			case reqInfo.Verb == "list":
				listHandler.ServeHTTP(w, req)
			case reqInfo.Verb == "watch":
				listHandler.ServeHTTP(w, req)
			default:
				responsewriters.ErrorNegotiated(errors.New("unsupport req verb"), ls.NegotiatedSerializer, schema.GroupVersion{}, w, req)
				return
			}
		} else {
			responsewriters.ErrorNegotiated(errors.New("not a resource req"), ls.NegotiatedSerializer, schema.GroupVersion{}, w, req)
		}
	})
	return h
}

func BuildHandlerChain(handler http.Handler, ls *MetaServer) http.Handler {
	cfg := &server.Config{
		LegacyAPIGroupPrefixes: sets.NewString(server.DefaultLegacyAPIPrefix),
	}
	//handler = genericfilters.WithTimeoutForNonLongRunningRequests(handler, ls.LongRunningFunc, ls.RequestTimeout)
	handler = genericfilters.WithWaitGroup(handler, ls.LongRunningFunc, ls.HandlerChainWaitGroup)
	handler = genericapifilters.WithRequestInfo(handler, server.NewRequestInfoResolver(cfg))
	//handler = genericapifilters.WithWarningRecorder(handler)
	//handler = genericapifilters.WithCacheControl(handler)
	handler = genericfilters.WithPanicRecovery(handler)
	return handler
}
