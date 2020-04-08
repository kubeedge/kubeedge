package streamcontroller

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/emicklei/go-restful"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/pkg/stream/flushwriter"
)

type Server struct {
	container *restful.Container
	tunnel    *TunnelServer
}

func newServer(t *TunnelServer) *Server {
	return &Server{
		container: restful.NewContainer(),
		tunnel:    t,
	}
}

func (s *Server) installDebugHandler() {
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

func (s *Server) getExec(r *restful.Request, w *restful.Response) {
	panic("unimplement")
}

func (s *Server) getContainerLogs(r *restful.Request, w *restful.Response) {

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

func (s *Server) Start() {
	s.installDebugHandler()

	tunnelServer := &http.Server{
		Addr:    ":10350",
		Handler: s.container,
	}
	//err := tunnelServer.ListenAndServeTLS("./server.crt", "./server.key")
	err := tunnelServer.ListenAndServe()
	if err != nil {
		klog.Fatalf("start server error %v\n", err)
	}
}
