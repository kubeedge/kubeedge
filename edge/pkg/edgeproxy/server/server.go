package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	proxyconfig "github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/config"
)

func NewProxyServer(eph http.Handler) (*ProxyServer, error) {
	return &ProxyServer{
		handler: buildHandleChain(eph),
		mux:     mux.NewRouter(),
	}, nil
}

func buildHandleChain(handler http.Handler) http.Handler {
	wrapper := WithReqContentType(handler)
	wrapper = WithAppUserAgent(wrapper)
	wrapper = WithRequestInfo(wrapper)
	return wrapper
}

type ProxyServer struct {
	mux     *mux.Router
	handler http.Handler
}

func (ps *ProxyServer) Run() {
	ps.installPath()
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(proxyconfig.Config.CaData)
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    certPool,
	}
	server := &http.Server{
		Handler:   ps.mux,
		Addr:      fmt.Sprintf(":%d", proxyconfig.Config.ListenPort),
		TLSConfig: tlsConfig,
	}
	err := server.ListenAndServeTLS(proxyconfig.Config.ServerCertFile, proxyconfig.Config.ServerKeyFile)
	if err != nil {
		panic(err)
	}
}
func (ps *ProxyServer) installPath() {
	ps.mux.HandleFunc("/healthz", ps.healthz).Methods("GET")
	ps.mux.PathPrefix("/").Handler(ps.handler)
}
func (ps *ProxyServer) healthz(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}
