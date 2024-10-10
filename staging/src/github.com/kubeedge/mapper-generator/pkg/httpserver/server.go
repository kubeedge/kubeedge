package httpserver

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"k8s.io/klog/v2"

	"github.com/kubeedge/mapper-generator/pkg/global"
)

type RestServer struct {
	IP             string
	Port           string
	WriteTimeout   time.Duration
	ReadTimeout    time.Duration
	CertFilePath   string
	KeyFilePath    string
	CaCertFilePath string
	server         *http.Server
	Router         *mux.Router
	devPanel       global.DevPanel
	databaseClient global.DataBaseClient
}

type Option func(server *RestServer)

func NewRestServer(devPanel global.DevPanel, options ...Option) *RestServer {
	rest := &RestServer{
		IP:           "0.0.0.0",
		Port:         "7777",
		Router:       mux.NewRouter(),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		devPanel:     devPanel,
	}
	for _, option := range options {
		option(rest)
	}
	return rest
}

func (rs *RestServer) StartServer() {
	rs.InitRouter()
	rs.server = &http.Server{
		Addr:         rs.IP + ":" + rs.Port,
		WriteTimeout: rs.WriteTimeout,
		ReadTimeout:  rs.ReadTimeout,
		Handler:      rs.Router,
	}
	if rs.CaCertFilePath == "" && (rs.KeyFilePath == "" || rs.CertFilePath == "") {
		// insecure
		klog.Info("Insecure communication, skipping server verification")
		err := rs.server.ListenAndServe()
		if err != nil {
			klog.Errorf("insecure http server error: %v", err)
			return
		}
	} else if rs.CaCertFilePath == "" && rs.KeyFilePath != "" && rs.CertFilePath != "" {
		// tls
		klog.Info("tls communication, https server start")
		err := rs.server.ListenAndServeTLS(rs.CertFilePath, rs.KeyFilePath)
		if err != nil {
			klog.Errorf("tls http server error: %v", err)
			return
		}
	} else if rs.CaCertFilePath != "" && rs.KeyFilePath != "" && rs.CertFilePath != "" {
		// mtls
		klog.Info("mtls communication, please provide client-key and client-cert to access service")
		// Configure the server to trust TLS client cert issued by your CA.
		certPool := x509.NewCertPool()
		if caCertPEM, err := ioutil.ReadFile(rs.CaCertFilePath); err != nil {
			klog.Errorf("Error loading ca certificate file: %v", err)
			return
		} else if ok := certPool.AppendCertsFromPEM(caCertPEM); !ok {
			klog.Error("invalid cert in CA PEM")
			return
		}
		tlsConfig := &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  certPool,
		}
		rs.server.TLSConfig = tlsConfig
		err := rs.server.ListenAndServeTLS(rs.CertFilePath, rs.KeyFilePath)
		if err != nil {
			klog.Errorf("tls http server error: %v", err)
			return
		}
	} else {
		klog.Error("the certificate file provided is incomplete or does not match")
	}
}

// sendResponse build response and put response's payload to writer
func (rs *RestServer) sendResponse(
	writer http.ResponseWriter,
	request *http.Request,
	response interface{},
	statusCode int) {

	correlationID := request.Header.Get(CorrelationHeader)
	if correlationID != "" {
		writer.Header().Set(CorrelationHeader, correlationID)
	}
	writer.Header().Set(ContentType, ContentTypeJSON)
	writer.WriteHeader(statusCode)
	data, err := json.Marshal(response)
	if err != nil {
		klog.Errorf("marshal %s response error: %v", request.URL.Path, err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
	_, err = writer.Write(data)
	if err != nil {
		klog.Errorf("write %s response error: %v", request.URL.Path, err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

func WithIP(ip string) Option {
	return func(server *RestServer) {
		server.IP = ip
	}
}

func WithPort(port string) Option {
	return func(server *RestServer) {
		server.Port = port
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(server *RestServer) {
		server.WriteTimeout = timeout
	}
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(server *RestServer) {
		server.ReadTimeout = timeout
	}
}

func WithCertFile(certPath string) Option {
	return func(server *RestServer) {
		server.CertFilePath = certPath
	}
}

func WithKeyFile(keyFilePath string) Option {
	return func(server *RestServer) {
		server.KeyFilePath = keyFilePath
	}
}

func WithCaCertFile(caCertPath string) Option {
	return func(server *RestServer) {
		server.CaCertFilePath = caCertPath
	}
}

func WithDbClient(dbClient global.DataBaseClient) Option {
	return func(server *RestServer) {
		server.databaseClient = dbClient
	}
}
