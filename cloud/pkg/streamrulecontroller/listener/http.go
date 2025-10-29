/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package listener

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/router/utils"
	streamruleConfig "github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/config"
)

const MaxMessageBytes = 12 * (1 << 20)

var (
	StreamruleHandlerInstance = &StreamruleHandler{}
	kubeClient                *kubernetes.Clientset
)

type StreamruleHandler struct {
	restTimeout time.Duration
	handlers    sync.Map
	port        int
	bindAddress string
}

func initKubeClient() {
	config, err := rest.InClusterConfig()
	if err != nil {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

		config, err = clientConfig.ClientConfig()
		if err != nil {
			fmt.Printf("warn: kube client init failed: %v\n", err)
			kubeClient = nil
			return
		}
	}
	kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("warn: kube client build failed: %v\n", err)
		kubeClient = nil
	}
}

func InitHandler() {
	initKubeClient()

	timeout := streamruleConfig.Config.RestTimeout
	if timeout <= 0 {
		timeout = 60
	}
	StreamruleHandlerInstance.restTimeout = time.Duration(timeout) * time.Second
	StreamruleHandlerInstance.bindAddress = streamruleConfig.Config.Address
	StreamruleHandlerInstance.port = int(streamruleConfig.Config.Port)
	if StreamruleHandlerInstance.port <= 0 {
		StreamruleHandlerInstance.port = 9445
	}
	klog.Infof("streamrulecontroller init: %v", StreamruleHandlerInstance)
}

func (sh *StreamruleHandler) Serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", sh.httpHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", sh.bindAddress, sh.port),
		Handler: mux,
	}
	klog.Infof("streamrule server listening in %d...", sh.port)
	if err := server.ListenAndServe(); err != nil {
		klog.Errorf("start streamrule endpoint failed, err: %v", err)
	}
}

func (sh *StreamruleHandler) AddListener(key interface{}, han Handle) {
	path, ok := key.(string)
	if !ok {
		return
	}
	sh.handlers.Store(path, han)
}

func (sh *StreamruleHandler) RemoveListener(key interface{}) {
	path, ok := key.(string)
	if !ok {
		return
	}
	sh.handlers.Delete(path)
}

func (sh *StreamruleHandler) matchedPath(uri string) (string, bool) {
	var candidateRes string
	sh.handlers.Range(func(key, value interface{}) bool {
		pathReg, ok := key.(string)
		if !ok {
			klog.Errorf("key type %T error", key)
			return true
		}
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

func (sh *StreamruleHandler) httpHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Transfer-Encoding", "chunked")
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

	matchPath, exist := sh.matchedPath(r.RequestURI)
	if !exist {
		klog.Warningf("URL format incorrect: %s", r.RequestURI)
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("Request error")); err != nil {
			klog.Errorf("Response write error: %s, %s", r.RequestURI, err.Error())
		}
		return
	}
	v, ok := sh.handlers.Load(matchPath)

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
	b, err := io.ReadAll(aReaderCloser)
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
		params["timeout"] = sh.restTimeout
		params["data"] = b

		v, err := handle(params)
		if err != nil {
			klog.Errorf("handle request error, msg id: %s, err: %v", msgID, err)
			return
		}
		response, ok := v.(*http.Response)
		if !ok {
			klog.Errorf("response convert error, msg id: %s", msgID)
			return
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, MaxMessageBytes))
		if err != nil {
			klog.Errorf("response body read error, msg id: %s, reason: %v", msgID, err)
			return
		}
		for key, values := range response.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(response.StatusCode)
		if _, err = w.Write(body); err != nil {
			klog.Errorf("response body write error, msg id: %s, reason: %v", msgID, err)
			return
		}
		klog.Infof("response to client, msg id: %s, write result: success", msgID)
	} else {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("No streamrule match"))
		klog.Infof("no streamrule match, write result: %v", err)
	}
}

func isNodeName(str string) bool {
	if kubeClient == nil {
		// Always returns true when the kube client has not been initialized successfully
		klog.Warningf("kube client is nil")
		return true
	}

	node, err := kubeClient.CoreV1().Nodes().Get(context.TODO(), str, metav1.GetOptions{})
	if err != nil {
		return true
	}
	return node != nil
}
