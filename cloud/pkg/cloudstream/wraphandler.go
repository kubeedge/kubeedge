/*
Copyright 2020 The KubeEdge Authors.

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

package cloudstream

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/httpstream"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/klog/v2"
)

// WarpHandler intercepts http/https requests sent from the master, setup proxy tunnel to kubelet,
// sends requests through the tunnel and sends responses back to the master
type WarpHandler struct {
	UDSSockFile string
	TLSConfig   *tls.Config
}

func WrapHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		klog.V(1).Infof("start handling request %s %s, from %s to %s",
			req.Method, req.URL.String(), req.Host, req.RemoteAddr)
		start := time.Now()
		handler.ServeHTTP(w, req)
		klog.V(1).Infof("stop handling request %s %s, request handling lasts %v",
			req.Method, req.URL.String(), time.Since(start))
	})
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func cloneRequest(req *http.Request, scheme, host, path string) *http.Request {
	// shallow copy request
	r := new(http.Request)
	*r = *req
	// deep copy header, and url
	copyHeader(r.Header, req.Header)
	r.URL.Scheme = scheme
	r.URL.Host = host
	r.URL.Path = path
	r.RequestURI = ""
	utilnet.AppendForwardedForHeader(r)
	return r
}

func klogAndHTTPError(w http.ResponseWriter, errCode int, format string, i ...interface{}) {
	errMsg := fmt.Sprintf(format, i...)
	klog.Error(errMsg)
	http.Error(w, errMsg, errCode)
}

// setupTunnel sets up proxy tunnels from interceptor to kubelet
// i.e., interceptor <-> proxy <-> agent <-> kubelet
func setupTunnel(reqScheme, proxyAddr, destAddr string, tlsConfig *tls.Config) (net.Conn, error) {
	proxyConn, err := net.Dial("unix", proxyAddr)
	if err != nil {
		errMsg := fmt.Sprintf("fail to setup TCP connection to server: %s", err)
		klog.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// send CONNECT request to proxy server
	fmt.Fprintf(proxyConn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", destAddr, "127.0.0.1")
	br := bufio.NewReader(proxyConn)
	res, err := http.ReadResponse(br, nil)
	if err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("reading HTTP response from CONNECT to %s via proxy %s failed: %v",
			destAddr, proxyAddr, err)
	}
	if res.StatusCode != 200 {
		proxyConn.Close()
		return nil, fmt.Errorf("proxy error from %s while dialing %s, code %d: %v",
			proxyAddr, destAddr, res.StatusCode, res.Status)
	}
	klog.V(1).Info("successfully setup the proxy tunnel")

	return proxyConn, nil
}

func transfer(dest io.WriteCloser, src io.ReadCloser) {
	defer dest.Close()
	defer src.Close()
	io.Copy(dest, src)
}

func (wh *WarpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	klog.V(1).Infof("intercept a %s request from apiserver %s", scheme, r.RemoteAddr)
	newReq := cloneRequest(r, scheme, r.Host, r.URL.Path)
	wh.TLSConfig.InsecureSkipVerify = true

	tlsTunnelConn, err := setupTunnel(scheme, wh.UDSSockFile, r.Host, wh.TLSConfig)
	if err != nil {
		klogAndHTTPError(w, http.StatusServiceUnavailable,
			"fail to setup the tunnel: %s", err)
		return
	}

	if err := r.Write(tlsTunnelConn); err != nil {
		tlsTunnelConn.Close()
		klogAndHTTPError(w, http.StatusServiceUnavailable,
			"fail to write request to tls connection: %s", err)
		return
	}

	if httpstream.IsUpgradeRequest(r) {
		serveUpgradeRequest(tlsTunnelConn, w, newReq)
		return
	}

	serveRequest(tlsTunnelConn, w, newReq)
}

// serveUpgradeRequest serves the request that needs to be upgraded
// i.e. request requires bidirection httpstreaming
func serveUpgradeRequest(tlsTunnelConn net.Conn, w http.ResponseWriter, r *http.Request) {
	klog.V(1).Infof("start serving streaming request\n Headers: %v", r.Header)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		klogAndHTTPError(w, http.StatusServiceUnavailable,
			"failed to assert response to http.Hijacker")
		return
	}
	masterConn, _, err := hijacker.Hijack()
	if err != nil {
		klogAndHTTPError(w, http.StatusServiceUnavailable,
			"failed to hijack response: %s", err)
		return
	}
	// As we hijack the ResponseWriter, enable the keepalive mechanism for the
	// hijacked TCP connection.
	if tc, ok := masterConn.(*net.TCPConn); ok {
		if err := tc.SetKeepAlive(true); err != nil {
			klogAndHTTPError(w, http.StatusServiceUnavailable,
				"failed to enable keepalive: %s", err)
		}

		keepalivePeriod := 30 * time.Second
		if err := tc.SetKeepAlivePeriod(keepalivePeriod); err != nil {
			klogAndHTTPError(w, http.StatusServiceUnavailable,
				"failed to set keepalive period: %s", err)
		}
	}

	readerComplete, writerComplete :=
		make(chan struct{}), make(chan struct{})

	go func() {
		transfer(tlsTunnelConn, masterConn)
		close(readerComplete)
	}()
	go func() {
		transfer(masterConn, tlsTunnelConn)
		close(writerComplete)
	}()

	select {
	case <-writerComplete:
	case <-readerComplete:
	}
	klog.Infof("stop serving streaming request\n Headers: %v", r.Header)
}

// serverRequest serves the normal requests, e.g., kubectl logs
func serveRequest(tlsTunnelConn net.Conn, w http.ResponseWriter, r *http.Request) {
	select {
	case <-r.Context().Done():
		klog.Error("connection closed by apiserver")
		return
	default:
		repFromTunnel, err := http.ReadResponse(bufio.NewReader(tlsTunnelConn), nil)
		if err != nil {
			klogAndHTTPError(w, http.StatusServiceUnavailable,
				"failed to read response from the tunnel: %v", err)
			return
		}
		klog.V(1).Info("successfully read the http response from the proxy tunnel")
		defer repFromTunnel.Body.Close()

		copyHeader(w.Header(), repFromTunnel.Header)
		w.WriteHeader(repFromTunnel.StatusCode)

		if _, err := io.Copy(w, repFromTunnel.Body); err != nil {
			klogAndHTTPError(w, http.StatusServiceUnavailable,
				"failed to copy response from the tunnel back to the client: %s", err)
			return
		}

		klog.V(1).Infof("stop serving request\n"+
			"Headers: %v", r.Header)
	}
}
