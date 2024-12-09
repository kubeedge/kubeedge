package metaserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	utilwaitgroup "k8s.io/apimachinery/pkg/util/waitgroup"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/request/bearertoken"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/server"
	genericfilters "k8s.io/apiserver/pkg/server/filters"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/kubeclientbridge"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/auth"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/certificate"
	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/handlerfactory"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/serializer"
	kefeatures "github.com/kubeedge/kubeedge/pkg/features"
	passthrough "github.com/kubeedge/kubeedge/pkg/util/pass-through"
)

// MetaServer is simplification of server.GenericAPIServer
type MetaServer struct {
	HandlerChainWaitGroup *utilwaitgroup.SafeWaitGroup
	LongRunningFunc       apirequest.LongRunningRequestCheck
	RequestTimeout        time.Duration
	Handler               http.Handler
	NegotiatedSerializer  runtime.NegotiatedSerializer
	Factory               *handlerfactory.Factory
	Auth                  *metaServerAuth
	// Handles Authentication Certificate
	serverCeriticateManager *certificate.ServerCertificateManager
}

type metaServerAuth struct {
	Authenticator authenticator.Request
	Authorizer    authorizer.Authorizer
}

func buildAuth() *metaServerAuth {
	newAuthorizer := rbac.New(
		&client.RoleGetter{},
		&client.RoleBindingLister{},
		&client.ClusterRoleGetter{},
		&client.ClusterRoleBindingLister{})

	allPublicKeys := []interface{}{}
	for _, keyfile := range metaserverconfig.Config.ServiceAccountKeyFiles {
		publicKeys, err := keyutil.PublicKeysFromFile(keyfile)
		if err != nil {
			klog.Errorf("Failed to load public key file %s: %v", keyfile, err)
			return nil
		}
		allPublicKeys = append(allPublicKeys, publicKeys...)
	}
	tokenAuthenticator := auth.JWTTokenAuthenticator(nil,
		metaserverconfig.Config.ServiceAccountIssuers, allPublicKeys, metaserverconfig.Config.APIAudiences,
		auth.NewValidator(client.NewGetterFromClient(kubeclientbridge.NewSimpleClientset(client.New()))))
	newAuthenticator := bearertoken.New(tokenAuthenticator)
	return &metaServerAuth{newAuthenticator, newAuthorizer}
}

func NewMetaServer() *MetaServer {
	ls := MetaServer{
		HandlerChainWaitGroup: new(utilwaitgroup.SafeWaitGroup),
		LongRunningFunc:       genericfilters.BasicLongRunningRequestCheck(sets.NewString("watch"), sets.NewString()),
		NegotiatedSerializer:  serializer.NewNegotiatedSerializer(),
		Factory:               handlerfactory.NewFactory(),
		Auth:                  buildAuth(),
	}
	return &ls
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

func (ls *MetaServer) startHTTPSServer(addr string, stopChan <-chan struct{}) {
	tlsConfig, err := ls.makeTLSConfig()
	if err != nil {
		panic(err)
	}

	h := ls.BuildBasicHandler()
	h = BuildHandlerChain(h, ls)
	s := http.Server{
		Addr:      addr,
		Handler:   h,
		TLSConfig: tlsConfig,
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
		err := ls.prepareServer()
		if err != nil {
			panic(err)
		}
		go ls.startHTTPSServer(metaserverconfig.Config.Server, stopChan)
		go ls.startHTTPSServer(metaserverconfig.Config.DummyServer, stopChan)
	} else {
		ls.startHTTPServer(stopChan)
	}
}

func (ls *MetaServer) BuildBasicHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		reqInfo, ok := apirequest.RequestInfoFrom(ctx)
		// klog.Infof("[metaserver]get a req(%v)(%v)", reqInfo.Path, reqInfo.Verb)
		// klog.Infof("[metaserver]get a req(\nPath:%v; \nVerb:%v; \nHeader:%+v)", reqInfo.Path, reqInfo.Verb, req.Header)
		if !ok {
			err := fmt.Errorf("invalid request")
			responsewriters.ErrorNegotiated(apierrors.NewInternalError(err), ls.NegotiatedSerializer, schema.GroupVersion{}, w, req)
			return
		}

		if reqInfo.IsResourceRequest {
			switch {
			case reqInfo.Verb == "get":
				if reqInfo.Subresource == "log" {
					ls.Factory.Logs(reqInfo).ServeHTTP(w, req)
				} else {
					ls.Factory.Get().ServeHTTP(w, req)
				}
			case reqInfo.Verb == "list", reqInfo.Verb == "watch":
				ls.Factory.List().ServeHTTP(w, req)
			case reqInfo.Verb == "create":
				if reqInfo.Name == "restart" {
					ls.Factory.Restart(reqInfo.Namespace).ServeHTTP(w, req)
				} else if reqInfo.Name == "confirm-upgrade" {
					ls.Factory.ConfirmUpgrade().ServeHTTP(w, req)
				} else if reqInfo.Subresource == "exec" {
					ls.Factory.Exec(reqInfo).ServeHTTP(w, req)
				} else {
					ls.Factory.Create(reqInfo).ServeHTTP(w, req)
				}
			case reqInfo.Verb == "delete":
				ls.Factory.Delete().ServeHTTP(w, req)
			case reqInfo.Verb == "update":
				ls.Factory.Update(reqInfo).ServeHTTP(w, req)
			case reqInfo.Verb == "patch":
				ls.Factory.Patch(reqInfo).ServeHTTP(w, req)
			default:
				err := fmt.Errorf("unsupported req verb")
				responsewriters.ErrorNegotiated(apierrors.NewInternalError(err), ls.NegotiatedSerializer, schema.GroupVersion{}, w, req)
			}
			return
		}

		if passthrough.IsPassThroughPath(reqInfo.Path, reqInfo.Verb) {
			ls.Factory.PassThrough().ServeHTTP(w, req)
			return
		}

		err := fmt.Errorf("request[%s::%s] isn't supported", reqInfo.Path, reqInfo.Verb)
		responsewriters.ErrorNegotiated(apierrors.NewInternalError(err), ls.NegotiatedSerializer, schema.GroupVersion{}, w, req)
	})
}

func BuildHandlerChain(handler http.Handler, ls *MetaServer) http.Handler {
	cfg := &server.Config{
		LegacyAPIGroupPrefixes: sets.NewString(server.DefaultLegacyAPIPrefix),
	}
	if kefeatures.DefaultFeatureGate.Enabled(kefeatures.RequireAuthorization) {
		handler = genericapifilters.WithAuthorization(handler, ls.Auth.Authorizer, legacyscheme.Codecs)
		failedHandler := genericapifilters.Unauthorized(legacyscheme.Codecs)
		handler = genericapifilters.WithAuthentication(handler, ls.Auth.Authenticator, failedHandler, metaserverconfig.Config.APIAudiences, nil)
	}
	handler = genericfilters.WithWaitGroup(handler, ls.LongRunningFunc, ls.HandlerChainWaitGroup)
	handler = genericapifilters.WithRequestInfo(handler, server.NewRequestInfoResolver(cfg))
	handler = genericfilters.WithPanicRecovery(handler, &apirequest.RequestInfoFactory{})
	return handler
}

func (ls *MetaServer) prepareServer() error {
	err := setupDummyInterface()
	if err != nil {
		return fmt.Errorf("setup dummy interface err: %v", err)
	}
	certIPs, err := ls.getCertIPs()
	if err != nil {
		return fmt.Errorf("failed to get cert IP: %v", err)
	}

	certificateManager, err := certificate.NewServerCertificateManager(
		certificate.NewSimpleClientset(),
		types.NodeName(metaserverconfig.Config.NodeName),
		certIPs,
		certificate.CertificatesDir)
	if err != nil {
		return fmt.Errorf("failed to initialize certificate manager: %v", err)
	}

	err = certificateManager.WaitForCAReady()
	if err != nil {
		return fmt.Errorf("wait for CA ready failed: %v", err)
	}

	certificateManager.Start()
	err = certificateManager.WaitForCertReady()
	if err != nil {
		return fmt.Errorf("wait for cert ready failed: %v", err)
	}

	ls.serverCeriticateManager = certificateManager
	return nil
}

func (ls *MetaServer) getCertIPs() ([]net.IP, error) {
	ip, _, err := net.SplitHostPort(metaserverconfig.Config.Server)
	if err != nil {
		return nil, err
	}
	dummyIP, _, err := net.SplitHostPort(metaserverconfig.Config.DummyServer)
	if err != nil {
		return nil, err
	}
	return []net.IP{net.ParseIP(ip), net.ParseIP(dummyIP)}, nil
}

func (ls *MetaServer) makeTLSConfig() (*tls.Config, error) {
	ca, err := os.ReadFile(fmt.Sprintf("%s/ca.crt", certificate.CertificatesDir))
	if err != nil {
		return nil, fmt.Errorf("read CA failed: %v", err)
	}

	block, _ := pem.Decode(ca)
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: block.Bytes}))
	if !ok {
		return nil, errors.New("failed to load ca content")
	}

	return &tls.Config{
		ClientCAs:  pool,
		MinVersion: tls.VersionTLS12,
		ClientAuth: tls.VerifyClientCertIfGiven,
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert := ls.serverCeriticateManager.Current()
			if cert == nil {
				return nil, fmt.Errorf("no serving certificate available for the kubelet")
			}
			return cert, nil
		},
	}, nil
}
