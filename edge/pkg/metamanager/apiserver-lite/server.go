package apiserver_lite

import (
	"bytes"
	"context"
	"fmt"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/apiserver-lite/handlerfactory"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/apiserver-lite/kubernetes/serializer"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/apiserver-lite/kubernetes/storage/sqlite/imitator"
	"github.com/kubeedge/kubeedge/pkg/apiserverlite"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	utilwaitgroup "k8s.io/apimachinery/pkg/util/waitgroup"
	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/server"
	genericfilters "k8s.io/apiserver/pkg/server/filters"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

const(
	// TODO: make addrs configurable
	httpaddr = "0.0.0.0:10010"
	apiserverAddr = "https://10.10.102.81:6443"
)
// TODO: is it necessary to construct a new struct rather than server.GenericAPIServer ?
// LiteServer is simplification of server.GenericAPIServer
type LiteServer struct{
	HandlerChainWaitGroup *utilwaitgroup.SafeWaitGroup
	LongRunningFunc apirequest.LongRunningRequestCheck
	RequestTimeout time.Duration
	Handler http.Handler
	NegotiatedSerializer runtime.NegotiatedSerializer
	Factory handlerfactory.Factory
	reverseProxy *httputil.ReverseProxy
}

func NewLiteServer() *LiteServer {
	apiserverURL, err := url.Parse(apiserverAddr)
	utilruntime.Must(err)

	// TODO: make kubeConfig configurable
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", "kubeconfig")
	utilruntime.Must(err)
	reverseProxy := httputil.NewSingleHostReverseProxy(apiserverURL)
	ls := LiteServer{
		HandlerChainWaitGroup:  new(utilwaitgroup.SafeWaitGroup),
		LongRunningFunc:        genericfilters.BasicLongRunningRequestCheck(sets.NewString("watch"), sets.NewString()),
		NegotiatedSerializer:   serializer.NewNegotiatedSerializer(),
		Factory: handlerfactory.NewFactory(),
		reverseProxy: reverseProxy,
	}
	transport,err:= rest.TransportFor(kubeConfig)
	reverseProxy.Transport = transport
	reverseProxy.FlushInterval = -1
	reverseProxy.ModifyResponse = ls.CacheGetResp
	return &ls
}
func(ls *LiteServer)CacheGetResp(resp *http.Response) error {
	if resp == nil || resp.Request == nil {
		klog.V(4).Infof("[apiserver-lite]cache resp, no resp or request info in response, skip cache get response")
		return nil
	}
	req := resp.Request
	ctx := req.Context()
	reqInfo, ok := apirequest.RequestInfoFrom(ctx)
	if !ok  {
		klog.V(4).Infof("[apiserver-lite]cache resp, no req info in req context, skip")
		return nil
	}
	klog.V(4).Infof("[apiserver-lite]cache resp, req header: %v ; resp header: %v.",req.Header, resp.Header)
	if reqInfo.Subresource == "status" {
		klog.V(4).Infof("[apiserver-lite]cache resp, no req info in req context, skip")
	}
	if reqInfo.IsResourceRequest && (reqInfo.Verb == "get" || reqInfo.Verb == "update"|| reqInfo.Verb == "insert" || reqInfo.Verb=="patch"){
		var buf bytes.Buffer
		n, err := buf.ReadFrom(resp.Body)
		if err !=nil || n == 0{
			return err
		}
		reader, writer := io.Pipe()
		resp.Body = reader
		go func(){
			_,err =writer.Write(buf.Bytes())
			if err !=nil{
				klog.Errorf("%v",err)
			}
			err = writer.Close()
			if err !=nil{
				klog.Errorf("%v",err)
			}
		}()
		mediaType, ok := negotiation.NegotiateMediaTypeOptions(resp.Header.Get("Accept"), ls.NegotiatedSerializer.SupportedMediaTypes(), negotiation.DefaultEndpointRestrictions)
		if !ok {
			return fmt.Errorf("cache resp, failed to negotiate")
		}
		serializerInfo := mediaType.Accepted
		var unstrObj unstructured.Unstructured
		_,_,err = serializerInfo.Serializer.Decode(buf.Bytes(),nil,&unstrObj)
		if err !=nil{
			return err
		}
		err = imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(),&unstrObj)
		if err !=nil{
			return err
		}
	}else{
		klog.V(4).Infof("[apiserver-lite]no need to cache resp",req.URL)
		return nil
	}
	klog.V(4).Infof("[apiserver-lite]successfully cache resp %v",req.URL)
	return nil

}

func (ls *LiteServer)Start(stopChan <-chan struct{}){
	h := ls.BuildBasicHandler()
	h = BuildHandlerChain(h,ls)
	s := http.Server{
		Addr:    httpaddr,
		Handler: h,
	}
	utilruntime.HandleError(s.ListenAndServe())
	<-stopChan
}

func(ls *LiteServer)BuildBasicHandler()http.Handler{
	listHandler := ls.Factory.List()
	h:= http.HandlerFunc(func(w http.ResponseWriter, req *http.Request){
		ctx := req.Context()
		reqInfo , ok := apirequest.RequestInfoFrom(ctx)
		klog.Infof("[apiserver-lite]get a req(%v):%+v \nreq:%+v",reqInfo.Path,reqInfo,req)
		if ok && reqInfo.IsResourceRequest{
			// try remote apiserver
			switch {
			case reqInfo.Resource == "secrets", reqInfo.Resource == "configmaps":
				if err := ls.tryRemote(w,req); err == nil{
					klog.Infof("successfull transport req to remote")
					return
				}
			case reqInfo.Verb == "get":
				if err := ls.tryRemote(w,req); err == nil{
					klog.Infof("successfull transport req to remote")
					return
				}
			default:
			}

			// try local, also transport req to remote apiserver if req type is not read opration.
			switch{
			case reqInfo.Verb == "get":
				// TODO: @rachel-shao, replace with getHandler.ServerHttp
				ls.processRead(w,req)
			case reqInfo.Verb == "list":
				listHandler.ServeHTTP(w,req)
			case reqInfo.Verb == "watch":
				listHandler.ServeHTTP(w,req)
			default:
				// temporarily force set to json
				req.Header.Set("Accept","application/json")
				ls.reverseProxy.ServeHTTP(w,req)
			}
		} else{
			ls.reverseProxy.ServeHTTP(w,req)
		}

	})
	return h
}

func(ls *LiteServer)tryRemote(w http.ResponseWriter,req *http.Request)error{
	conn, err := net.DialTimeout("tcp", "10.10.102.81:6443", 3*time.Second)
	if err ==nil && conn !=nil {
		// transport to apiserver
		// temporarily force set to json
		req.Header.Set("Accept","application/json")
		ls.reverseProxy.ServeHTTP(w,req)
		return nil
	}
	return fmt.Errorf("failed to transport req to remote, %v",err)
}

func(ls *LiteServer)processRead(w http.ResponseWriter,req *http.Request){
	ctx := req.Context()
	info, _  :=apirequest.RequestInfoFrom(ctx)
	key,err:= apiserverlite.KeyFuncReq(ctx,"")
	if err !=nil{
		responsewriters.InternalError(w,req,err)
		return
	}
	resp, err := imitator.DefaultV2Client.Get(context.TODO(),key)
	gv := schema.GroupVersion{
		Group: info.APIGroup,
		Version: info.APIVersion,
	}
	if err !=nil || len(*resp.Kvs) == 0{
		responsewriters.ErrorNegotiated(err,ls.NegotiatedSerializer,gv,w,req)
		//responsewriters.InternalError(w,req,err)
		klog.Error(err)
		return
	}
	mdeiaType,_,err:= negotiation.NegotiateOutputMediaType(req,ls.NegotiatedSerializer,negotiation.DefaultEndpointRestrictions)
	if err !=nil{
		responsewriters.ErrorNegotiated(err,ls.NegotiatedSerializer,gv,w,req)
		return
	}
	meta := (*resp.Kvs)[0]
	b := []byte(meta.Value)
	switch mdeiaType.Accepted.MediaType {
	case "application/json":
		writeRaw(http.StatusOK, b,w)
	case "application/yaml"://convert to yaml format
		var unstrObj unstructured.Unstructured
		_,_,err = unstructured.UnstructuredJSONScheme.Decode(b,nil,&unstrObj)
		if err!=nil{
			responsewriters.InternalError(w,req,err)
		}
		responsewriters.WriteObjectNegotiated(ls.NegotiatedSerializer,negotiation.DefaultEndpointRestrictions,gv,w,req,http.StatusOK,&unstrObj)
	default :
		klog.Errorf("do not support this, %v",mdeiaType.Accepted.MediaType)
	}
	return
}
func writeRaw(statusCode int,bytes []byte,w http.ResponseWriter){
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(bytes)
}

func(ls *LiteServer)processFail(w http.ResponseWriter,req *http.Request){
	//TODO: transfer to remote true apiserver

}
func BuildHandlerChain(handler http.Handler, ls *LiteServer) http.Handler {
	cfg := &server.Config{
		LegacyAPIGroupPrefixes: sets.NewString(server.DefaultLegacyAPIPrefix),
	}
	//handler = genericfilters.WithTimeoutForNonLongRunningRequests(handler, ls.LongRunningFunc, ls.RequestTimeout)
	handler = genericfilters.WithWaitGroup(handler, ls.LongRunningFunc, ls.HandlerChainWaitGroup)
	handler = genericapifilters.WithRequestInfo(handler,server.NewRequestInfoResolver(cfg))
	//handler = genericapifilters.WithWarningRecorder(handler)
	//handler = genericapifilters.WithCacheControl(handler)
	handler = genericfilters.WithPanicRecovery(handler)
	return handler
}

