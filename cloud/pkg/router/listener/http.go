package listener

import (
	"fmt"
	"github.com/google/uuid"
	routerConfig "github.com/kubeedge/kubeedge/cloud/pkg/router/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/utils"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	RestHandlerInstance = &RestHandler{}
)

type RestHandler struct {
	restTimeout time.Duration
	handlers    sync.Map
	port        int
	bindAddress string
}

func InitHandler() {
	timeout := routerConfig.Config.RestTimeout
	if timeout <= 0 {
		timeout = 60
	}
	RestHandlerInstance.restTimeout = time.Duration(timeout) * time.Second
	RestHandlerInstance.bindAddress = routerConfig.Config.Address
	RestHandlerInstance.port = int(routerConfig.Config.Port)
	if RestHandlerInstance.port <= 0 {
		RestHandlerInstance.port = 9443
	}
	klog.Infof("rest init: %v", RestHandlerInstance)
}

func (rh *RestHandler) Serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", rh.httpHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", rh.bindAddress, rh.port),
		Handler: mux,
		//TLSConfig: rc.GetTLSServerConfig(),
		//ErrorLog:  llog.New(&common.FilterWriter{}, "", llog.LstdFlags),
	}
	klog.Infof("listening in %d...", rh.port)
	//err := server.ListenAndServeTLS("", "")
	err := server.ListenAndServe()
	if err != nil {
		klog.Errorf("start rest endpoint failed, err: %v", err)
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
			if candidateRes != "" && utils.RuleContains(strings.Split(pathReg, "/"), strings.Split(candidateRes, "/")) {
				return true
			} else {
				candidateRes = pathReg
			}
		}
		return true
	})
	if candidateRes == "" {
		return "", false
	} else {
		return candidateRes, true
	}
}

func (rh *RestHandler) httpHandler(w http.ResponseWriter, r *http.Request) {
	uriSections := strings.Split(r.RequestURI, "/")
	if len(uriSections) < 2 {
		//URL format incorrect
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Request error"))
		klog.Warningf("URL format incorrect: %s", r.RequestURI)
		return
	}

	matchPath, exist := rh.matchedPath(r.RequestURI)
	if !exist {
		klog.Warningf("URL format incorrect: %s", r.RequestURI)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Request error"))
		return
	}
	v, ok := rh.handlers.Load(matchPath)
	if !ok {
		klog.Warningf("No matched handler for path: %s", matchPath)
		return
	}
	handle, ok := v.(Handle)
	if !ok {
		klog.Errorf("invalid conver to Handl. mathch path: %s", matchPath)
		return
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("request error, write result: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request error,body is null"))
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
			//w.WriteHeader(http.StatusInternalServerError)
			//_, err := w.Write([]byte(err.Error()))
			//klog.Warningf("operation timeout, msg id: %s, write result: %v", msgID, err)
			return
		}
		response, ok := v.(*http.Response)
		if !ok {

		}
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {

		}
		w.WriteHeader(response.StatusCode)
		w.Write(body)
		klog.Infof("response to client, msg id: %s, write result: %v", msgID, "success")
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

func isNodeName(str string) bool {
	//if isOk, _ := regexp.MatchString("[-a-z0-9]{36}", str); isOk {
	//	return true
	//}
	//return false
	return true
}
