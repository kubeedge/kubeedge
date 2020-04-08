package streamcontroller

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/emicklei/go-restful"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/pkg/streamcontroller/config"
	"github.com/kubeedge/kubeedge/pkg/stream/flushwriter"
)

type StreamServer struct {
	container *restful.Container
	tunnel    *TunnelServer
}

func newStreamServer(t *TunnelServer) *StreamServer {
	return &StreamServer{
		container: restful.NewContainer(),
		tunnel:    t,
	}
}

func (s *StreamServer) installDebugHandler() {
	ws := new(restful.WebService)
	ws.Path("/containerLogs")
	ws.Route(ws.GET("/{podNamespace}/{podID}/{containerName}").
		To(s.getContainerLogs))
	s.container.Add(ws)

	ws = new(restful.WebService)
	ws.Path("/exec")

	ws.Route(ws.GET("/{podNamespace}/{podID}/{containerName}").
		To(s.getExec))
	ws.Route(ws.POST("/{podNamespace}/{podID}/{containerName}").
		To(s.getExec))
	ws.Route(ws.GET("/{podNamespace}/{podID}/{uid}/{containerName}").
		To(s.getExec))
	ws.Route(ws.POST("/{podNamespace}/{podID}/{uid}/{containerName}").
		To(s.getExec))
	s.container.Add(ws)
}

func (s *StreamServer) getExec(r *restful.Request, w *restful.Response) {
	panic("unimplement")
}

func (s *StreamServer) getContainerLogs(r *restful.Request, w *restful.Response) {

	hostKey := r.HeaderParameter("Host")
	session, ok := s.tunnel.getSession(hostKey)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		klog.Errorf("can not find %v session ", hostKey)
		return
	}

	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	if _, ok := w.ResponseWriter.(http.Flusher); !ok {
		w.WriteError(http.StatusInternalServerError,
			fmt.Errorf("unable to convert %v into http.Flusher, cannot show logs", reflect.TypeOf(w)))
		return
	}
	fw := flushwriter.Wrap(w.ResponseWriter)

	logConnection, err := session.AddAPIServerConnection(&LogsConnection{
		r:       r,
		flush:   fw,
		session: session})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		klog.Errorf("create apiserver connection error %v", err)
		return
	}

	if err := logConnection.Serve(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		klog.Errorf("apiconnection Serve error %v", err)
		return
	}
}

func (s *StreamServer) Start() {
	s.installDebugHandler()

	pool := x509.NewCertPool()
	data, err := ioutil.ReadFile(config.Config.TLSStreamCAFile)
	if err != nil {
		klog.Fatalf("read tls stream ca file error %v", err)
	}
	pool.AppendCertsFromPEM(data)

	tunnelServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Config.StreamPort),
		Handler: s.container,
		TLSConfig: &tls.Config{
			ClientCAs: pool,
		},
	}

	err = tunnelServer.ListenAndServeTLS(config.Config.TLSStreamCertFile, config.Config.TLSStreamPrivateKeyFile)
	if err != nil {
		klog.Fatalf("start stream server error %v\n", err)
	}
}
