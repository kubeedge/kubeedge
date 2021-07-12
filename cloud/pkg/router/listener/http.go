package listener

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	hubConfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	hubHttpServer "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/httpserver"
	routerConfig "github.com/kubeedge/kubeedge/cloud/pkg/router/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/utils"
)

const MaxMessageBytes = 12 * (1 << 20)

var (
	RestHandlerInstance = &RestHandler{}
)

type RestHandler struct {
	restTimeout time.Duration
	handlers    sync.Map
	port        int
	bindAddress string
	securePort  int
}

func InitHandler() {
	timeout := routerConfig.Config.RestTimeout
	if timeout <= 0 {
		timeout = 60
	}
	RestHandlerInstance.restTimeout = time.Duration(timeout) * time.Second
	RestHandlerInstance.bindAddress = routerConfig.Config.Address
	RestHandlerInstance.port = int(routerConfig.Config.Port)
	RestHandlerInstance.securePort = int(routerConfig.Config.SecurePort)

	klog.Infof("rest init: %v", RestHandlerInstance)
}

func (rh *RestHandler) Serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", rh.httpHandler)

	// If Config.Port set to 0,the secure port is closed
	if int(routerConfig.Config.Port) != 0 {
		go startInSecureServer(rh)
	}

	// If set to 0 , the secure port is closed
	if int(routerConfig.Config.SecurePort) != 0 {
		// TODO: Will improve in the future
		ok := <-cloudhub.DoneTLSRouterCerts
		if ok {
			go startSecureServer(rh)
		}
	}
}

func startInSecureServer(rh *RestHandler) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", rh.httpHandler)
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", rh.bindAddress, rh.port),
		Handler: mux,
	}
	klog.Infof("Router server listening in %d...", rh.port)
	if err := server.ListenAndServe(); err != nil {
		klog.Errorf("Start rest endpoint failed, err: %v", err)
	}
}

func startSecureServer(rh *RestHandler) {
	var data []byte
	var key []byte
	var cert []byte
	if routerConfig.Config.Ca != nil {
		data = routerConfig.Config.Ca
		klog.Info("Succeed in loading RouterCA from local directory")
	} else {
		data = hubConfig.Config.Ca
		klog.Info("Succeed in loading RouterCA from CloudHub")
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: data}))

	if routerConfig.Config.Key != nil && routerConfig.Config.Cert != nil {
		cert = routerConfig.Config.Cert
		key = routerConfig.Config.Key
		klog.Info("Succeed in loading RouterCert and Key from local directory")
	} else {
		klog.Info("Router's Cert and key don't exist in the path, and will be signed by Cloudhub's CA")
		certDER, keyDER, err := hubHttpServer.SignCerts()
		if err != nil {
			klog.Errorf("Failed to sign router's certificate, error: %v", err)
		}
		cert = certDER
		key = keyDER
		klog.Info("Succeed in loading RouterCert and Key from CloudHub")
	}

	certificate, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: cert}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}))
	if err != nil {
		klog.Error("Failed to load TLSRouterCert and Key")
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", rh.httpHandler)
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", rh.bindAddress, rh.securePort),
		Handler: mux,
		TLSConfig: &tls.Config{
			ClientCAs:    pool,
			Certificates: []tls.Certificate{certificate},
			ClientAuth:   tls.RequestClientCert,
			MinVersion:   tls.VersionTLS12,
			CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
		},
	}

	klog.Infof("Router server listening in secure port %d...", rh.securePort)
	if err := server.ListenAndServeTLS("", ""); err != nil {
		klog.Errorf("Start secure rest endpoint failed, err: %v", err)
	}
}

func (rh *RestHandler) AddListener(key interface{}, han Handle) {
	path, ok := key.(string)
	if !ok {
		return
	}

	rh.handlers.Store(path, han)
}

func (rh *RestHandler) RemoveListener(key interface{}) {
	path, ok := key.(string)
	if !ok {
		return
	}
	rh.handlers.Delete(path)
}

func (rh *RestHandler) matchedPath(uri string) (string, bool) {
	var candidateRes string
	rh.handlers.Range(func(key, value interface{}) bool {
		pathReg := key.(string)
		if match := utils.IsMatch(pathReg, uri); match {
			if candidateRes != "" && utils.RuleContains(pathReg, candidateRes) {
				return true
			}
			candidateRes = pathReg
		}
		return true
	})
	if candidateRes == "" {
		return "", false
	}
	return candidateRes, true
}

func (rh *RestHandler) httpHandler(w http.ResponseWriter, r *http.Request) {
	uriSections := strings.Split(r.RequestURI, "/")
	if len(uriSections) < 2 {
		// URL format incorrect
		klog.Warningf("url format incorrect: %s", r.URL.String())
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("Request error")); err != nil {
			klog.Errorf("Response write error: %s, %s", r.RequestURI, err.Error())
		}
		return
	}

	matchPath, exist := rh.matchedPath(r.RequestURI)
	if !exist {
		klog.Warningf("URL format incorrect: %s", r.RequestURI)
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("Request error")); err != nil {
			klog.Errorf("Response write error: %s, %s", r.RequestURI, err.Error())
		}
		return
	}
	v, ok := rh.handlers.Load(matchPath)
	if !ok {
		klog.Warningf("No matched handler for path: %s", matchPath)
		return
	}
	handle, ok := v.(Handle)
	if !ok {
		klog.Errorf("invalid convert to Handle. match path: %s", matchPath)
		return
	}
	aReaderCloser := http.MaxBytesReader(w, r.Body, MaxMessageBytes)
	b, err := ioutil.ReadAll(aReaderCloser)
	if err != nil {
		klog.Errorf("request error, write result: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		if _, err = w.Write([]byte("Request error,body is null")); err != nil {
			klog.Errorf("Response write error: %s, %s", r.RequestURI, err.Error())
		}
		return
	}

	if isNodeName(uriSections[1]) {
		params := make(map[string]interface{})
		msgID := uuid.New().String()
		params["messageID"] = msgID
		params["request"] = r
		params["timeout"] = rh.restTimeout
		params["data"] = b

		v, err := handle(params)
		if err != nil {
			klog.Errorf("handle request error, msg id: %s, err: %v", msgID, err)
			return
		}
		response, ok := v.(*http.Response)
		if !ok {
			klog.Errorf("response convert error, msg id: %s, reason: %v", msgID, err)
			return
		}
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			klog.Errorf("response body read error, msg id: %s, reason: %v", msgID, err)
			return
		}
		w.WriteHeader(response.StatusCode)
		if _, err = w.Write(body); err != nil {
			klog.Errorf("response body write error, msg id: %s, reason: %v", msgID, err)
			return
		}
		klog.Infof("response to client, msg id: %s, write result: success", msgID)
	} else {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("No rule match"))
		klog.Infof("no rule match, write result: %v", err)
	}
}

func (rh *RestHandler) IsMatch(key interface{}, message interface{}) bool {
	res, ok := key.(string)
	if !ok {
		return false
	}
	uri, ok := message.(string)
	if !ok {
		return false
	}
	return utils.IsMatch(res, uri)
}

// TODO: check node name
func isNodeName(str string) bool {
	return true
}
