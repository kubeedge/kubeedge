package listener

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	routerConfig "github.com/kubeedge/kubeedge/cloud/pkg/router/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/utils"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/util"
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
		// TODO: add tls for router
	}
	klog.Infof("router server listening in %d...", rh.port)
	//err := server.ListenAndServeTLS("", "")
	if err := server.ListenAndServe(); err != nil {
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

func (rh *RestHandler) httpHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Transfer-Encoding", "chunked")
	uriSections := strings.Split(r.RequestURI, "/")
	if len(uriSections) < 2 {
		// URL format incorrect
		err := fmt.Errorf("url format incorrect: %s", r.URL.String())
		writeErr(w, r, http.StatusNotFound, err)
		return
	}

	aReaderCloser := http.MaxBytesReader(w, r.Body, MaxMessageBytes)
	b, err := io.ReadAll(aReaderCloser)
	if err != nil {
		writeErr(w, r, http.StatusBadRequest, err)
		return
	}

	edgeNodeName := uriSections[1]
	err = retry.Do(
		func() error {
			targetCloudCoreIP, err := GetEdgeToCloudCoreIP(r.Context(), edgeNodeName)
			if err != nil {
				return err
			}

			hostnameOverride := util.GetHostname()
			localIP, err := util.GetLocalIP(hostnameOverride)
			if err != nil {
				return fmt.Errorf("failed to get cloudcore localIP with err:%v", err)
			}
			if targetCloudCoreIP != localIP {
				var url string
				if r.TLS != nil {
					url = "https://" + targetCloudCoreIP
				} else {
					url = "http://" + targetCloudCoreIP
				}
				url += ":" + strconv.Itoa(rh.port) + r.RequestURI
				reqBody := io.NopCloser(bytes.NewBuffer(b))
				forwardReq, err := http.NewRequest(r.Method, url, reqBody)
				if err != nil {
					return fmt.Errorf("failed to create forward request: %v", err)
				}

				forwardReq.TLS = r.TLS
				forwardReq.Header = make(http.Header)
				for key, values := range r.Header {
					forwardReq.Header[key] = values
				}
				return requestForward(targetCloudCoreIP, w, forwardReq)
			}

			matchPath, exist := rh.matchedPath(r.RequestURI)
			if !exist {
				klog.Warningf("URL format incorrect: %s", r.RequestURI)
				w.WriteHeader(http.StatusNotFound)
				if _, err := w.Write([]byte("Request error")); err != nil {
					klog.Errorf("Response write error: %s, %s", r.RequestURI, err.Error())
				}
				return nil
			}
			v, ok := rh.handlers.Load(matchPath)
			if !ok {
				klog.Warningf("No matched handler for path: %s", matchPath)
				return nil
			}
			handle, ok := v.(Handle)
			if !ok {
				klog.Errorf("invalid convert to Handle. match path: %s", matchPath)
				return nil
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
					return nil
				}
				response, ok := v.(*http.Response)
				if !ok {
					klog.Errorf("response convert error, msg id: %s", msgID)
					return nil
				}
				body, err := io.ReadAll(io.LimitReader(response.Body, MaxMessageBytes))
				if err != nil {
					klog.Errorf("response body read error, msg id: %s, reason: %v", msgID, err)
					return nil
				}
				for key, values := range response.Header {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}

				if response.StatusCode != http.StatusOK {
					errMsg := string(body)
					return errors.New(errMsg)
				}

				w.WriteHeader(response.StatusCode)
				if _, err = w.Write(body); err != nil {
					klog.Errorf("response body write error, msg id: %s, reason: %v", msgID, err)
					return nil
				}
				klog.Infof("response to client, msg id: %s, write result: success", msgID)
				return nil
			}
			w.WriteHeader(http.StatusNotFound)
			_, err = w.Write([]byte("No rule match"))
			klog.Infof("no rule match, write result: %v", err)
			return nil
		},
		retry.Delay(1*time.Second),
		retry.Attempts(3),
		retry.DelayType(retry.FixedDelay),
	)

	if err != nil {
		writeErr(w, r, http.StatusInternalServerError, err)
		return
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
func isNodeName(_ string) bool {
	return true
}

func GetEdgeToCloudCoreIP(ctx context.Context, nodeName string) (string, error) {
	node, err := client.GetKubeClient().CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get node:%s,err:%v", nodeName, err)
	}
	cloudCoreIP, ok := node.Annotations[constants.EdgeMappingCloudKey]
	if !ok {
		return "", fmt.Errorf("no corresponding cloudcore was found for edgeNode:%s", nodeName)
	}
	return cloudCoreIP, nil
}

func requestForward(targetCloudCoreIP string, w http.ResponseWriter, forwardReq *http.Request) error {
	httpClient := &http.Client{}
	resp, err := httpClient.Do(forwardReq)
	if err != nil {
		return fmt.Errorf("failed to forward request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			klog.Errorf("failed to close resp.Body with err:%v", err)
		}
	}(resp.Body)

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading body:%v", err)
		}
		errMsg := string(bodyBytes)
		return errors.New(errMsg)
	}

	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy resp.Body to writer with err:%v", err)
	}

	klog.Infof("forwarded request to %s successfully", targetCloudCoreIP)
	return nil
}

func writeErr(w http.ResponseWriter, r *http.Request, statusCode int, err error) {
	klog.Error(err.Error())
	w.WriteHeader(statusCode)
	if _, err := w.Write([]byte(err.Error())); err != nil {
		klog.Errorf("Response write error: %s, %s", r.RequestURI, err.Error())
	}
}
