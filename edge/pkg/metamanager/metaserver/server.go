package metaserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
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
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/handlerfactory"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/serializer"
	kefeatures "github.com/kubeedge/kubeedge/pkg/features"
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

func createTLSConfig() tls.Config {
	ca, err := os.ReadFile(metaserverconfig.Config.TLSCaFile)
	if err == nil {
		block, _ := pem.Decode(ca)
		ca = block.Bytes
		klog.Info("Succeed in loading CA certificate from local directory")
	}
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: ca}))
	if !ok {
		panic(fmt.Errorf("fail to load ca content"))
	}
	cert, err := os.ReadFile(metaserverconfig.Config.TLSCertFile)
	if err == nil {
		block, _ := pem.Decode(cert)
		cert = block.Bytes
		klog.Info("Succeed in loading certificate from local directory")
	}
	key, err := os.ReadFile(metaserverconfig.Config.TLSPrivateKeyFile)
	if err == nil {
		block, _ := pem.Decode(key)
		key = block.Bytes
		klog.Info("Succeed in loading private key from local directory")
	}

	certificate, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: cert}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}))
	if err != nil {
		panic(err)
	}
	return tls.Config{
		ClientCAs:    pool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
	}
}

// getCurrent returns current meta server certificate
func (ls *MetaServer) getCurrent() (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(metaserverconfig.Config.TLSCertFile, metaserverconfig.Config.TLSPrivateKeyFile)
	if err != nil {
		return nil, err
	}
	certs, err := x509.ParseCertificates(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate data: %v", err)
	}
	cert.Leaf = certs[0]
	return &cert, nil
}

func (ls *MetaServer) startHTTPServer(stopChan <-chan struct{}) {
	h := ls.BuildBasicHandler()
	h = BuildHandlerChain(h, ls)
	s := http.Server{
		Addr:    metaserverconfig.Config.Server,
		Handler: h,
	}

	go func() {
		<-stopChan

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.Shutdown(ctx); err != nil {
			klog.Errorf("Server shutdown failed: %s", err)
		}
	}()

	klog.Infof("[metaserver]start to listen and server at http://%v", s.Addr)
	utilruntime.HandleError(s.ListenAndServe())
	// When the MetaServer stops abnormally, other module services are stopped at the same time.
	beehiveContext.Cancel()
}

func (ls *MetaServer) startHTTPSServer(stopChan <-chan struct{}) {
	_, err := ls.getCurrent()
	if err != nil {
		// wait for cert created
		klog.Infof("[metaserver]waiting for cert created")
		<-edgehub.GetCertSyncChannel()[modules.MetaManagerModuleName]
	}

	h := ls.BuildBasicHandler()
	h = BuildHandlerChain(h, ls)
	tlsConfig := createTLSConfig()
	s := http.Server{
		Addr:      metaserverconfig.Config.Server,
		Handler:   h,
		TLSConfig: &tlsConfig,
	}

	go func() {
		<-stopChan

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.Shutdown(ctx); err != nil {
			klog.Errorf("Server shutdown failed: %s", err)
		}
	}()

	klog.Infof("[metaserver]start to listen and server at https://%v", s.Addr)
	utilruntime.HandleError(s.ListenAndServeTLS("", ""))
	// When the MetaServer stops abnormally, other module services are stopped at the same time.
	beehiveContext.Cancel()
}

func (ls *MetaServer) Start(stopChan <-chan struct{}) {
	if kefeatures.DefaultFeatureGate.Enabled(kefeatures.RequireAuthorization) {
		ls.startHTTPSServer(stopChan)
	} else {
		ls.startHTTPServer(stopChan)
	}
}

func (ls *MetaServer) BuildBasicHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
			}
			return
		}

		err := fmt.Errorf("not a resource req")
		responsewriters.ErrorNegotiated(errors.NewInternalError(err), ls.NegotiatedSerializer, schema.GroupVersion{}, w, req)
	})
}

func BuildHandlerChain(handler http.Handler, ls *MetaServer) http.Handler {
	cfg := &server.Config{
		LegacyAPIGroupPrefixes: sets.NewString(server.DefaultLegacyAPIPrefix),
	}

	handler = genericfilters.WithWaitGroup(handler, ls.LongRunningFunc, ls.HandlerChainWaitGroup)
	handler = genericapifilters.WithRequestInfo(handler, server.NewRequestInfoResolver(cfg))
	handler = genericfilters.WithPanicRecovery(handler, &apirequest.RequestInfoFactory{})
	handler = WithAuthorizationHeader(handler)
	return handler
}

func WithAuthorizationHeader(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		token := request.Header.Get(commontypes.AuthorizationKey)
		request = request.WithContext(context.WithValue(request.Context(), commontypes.AuthorizationKey, token))
		handler.ServeHTTP(writer, request)
	})
}
