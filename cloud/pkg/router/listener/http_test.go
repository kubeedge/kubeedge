package listener

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/agiledragon/gomonkey/v2"

	"github.com/kubeedge/kubeedge/pkg/util"
)

const localCloudCoreIP = "10.0.0.1"

func patchCloudCoreIPs(t *testing.T, targets ...string) *gomonkey.Patches {
	t.Helper()
	var calls int32
	patches := gomonkey.ApplyFunc(GetEdgeToCloudCoreIP, func(context.Context, string) (string, error) {
		i := int(atomic.AddInt32(&calls, 1)) - 1
		if i >= len(targets) {
			i = len(targets) - 1
		}
		return targets[i], nil
	})
	patches.ApplyFunc(util.GetLocalIP, func(string) (string, error) {
		return localCloudCoreIP, nil
	})
	return patches
}

func serverPort(t *testing.T, rawURL string) int {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url %q: %v", rawURL, err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("parse port %q: %v", u.Port(), err)
	}
	return port
}

func TestHTTPHandlerPassesThroughLocalResponseOnce(t *testing.T) {
	patches := patchCloudCoreIPs(t, localCloudCoreIP)
	defer patches.Reset()

	for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusNoContent, http.StatusBadRequest, http.StatusRequestTimeout} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			rh := &RestHandler{}
			var calls int32
			rh.AddListener("/default/api", func(interface{}) (interface{}, error) {
				atomic.AddInt32(&calls, 1)
				header := http.Header{}
				header.Set("X-Edge", "value")
				return &http.Response{
					StatusCode: code,
					Header:     header,
					Body:       io.NopCloser(strings.NewReader("edge body")),
				}, nil
			})

			rec := httptest.NewRecorder()
			rh.httpHandler(rec, httptest.NewRequest(http.MethodPost, "/node1/default/api", strings.NewReader("{}")))

			if got := atomic.LoadInt32(&calls); got != 1 {
				t.Fatalf("handler invoked %d times, want 1", got)
			}
			if rec.Code != code {
				t.Fatalf("status = %d, want %d", rec.Code, code)
			}
			if got := rec.Header().Values("X-Edge"); len(got) != 1 || got[0] != "value" {
				t.Fatalf("X-Edge = %q, want [\"value\"]", got)
			}
			if code != http.StatusNoContent && rec.Body.String() != "edge body" {
				t.Fatalf("body = %q, want %q", rec.Body.String(), "edge body")
			}
		})
	}
}

func TestHTTPHandlerPassesThroughForwardedResponseOnce(t *testing.T) {
	var calls int32
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != "{}" {
			t.Errorf("forwarded body = %q, want %q", body, "{}")
		}
		if got := r.Header.Get("X-Client"); got != "value" {
			t.Errorf("forwarded X-Client = %q, want %q", got, "value")
		}
		w.Header().Set("X-Peer", "value")
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte("peer body"))
	}))
	defer peer.Close()

	patches := patchCloudCoreIPs(t, "127.0.0.1")
	defer patches.Reset()

	rh := &RestHandler{port: serverPort(t, peer.URL)}
	req := httptest.NewRequest(http.MethodPost, "/node1/default/api", strings.NewReader("{}"))
	req.Header.Set("X-Client", "value")
	rec := httptest.NewRecorder()
	rh.httpHandler(rec, req)

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("peer received %d requests, want 1", got)
	}
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusGatewayTimeout)
	}
	if got := rec.Header().Values("X-Peer"); len(got) != 1 || got[0] != "value" {
		t.Fatalf("X-Peer = %q, want [\"value\"]", got)
	}
	if rec.Body.String() != "peer body" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "peer body")
	}
}

func TestHTTPHandlerRetriesUnreachablePeerCloudCore(t *testing.T) {
	peer := httptest.NewServer(http.NotFoundHandler())
	port := serverPort(t, peer.URL)
	peer.Close()

	patches := patchCloudCoreIPs(t, "127.0.0.1", localCloudCoreIP)
	defer patches.Reset()

	rh := &RestHandler{port: port}
	var calls int32
	rh.AddListener("/default/api", func(interface{}) (interface{}, error) {
		atomic.AddInt32(&calls, 1)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	})

	rec := httptest.NewRecorder()
	rh.httpHandler(rec, httptest.NewRequest(http.MethodPost, "/node1/default/api", strings.NewReader("{}")))

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("handler invoked %d times, want 1", got)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "ok")
	}
}

func TestHTTPHandlerDoesNotRetryPeerCloudCoreAfterRequestIsSent(t *testing.T) {
	var calls int32
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("Hijack returned error: %v", err)
			return
		}
		if err := conn.Close(); err != nil {
			t.Errorf("Close returned error: %v", err)
		}
	}))
	defer peer.Close()

	patches := patchCloudCoreIPs(t, "127.0.0.1")
	defer patches.Reset()

	rh := &RestHandler{port: serverPort(t, peer.URL)}
	rec := httptest.NewRecorder()
	rh.httpHandler(rec, httptest.NewRequest(http.MethodPost, "/node1/default/api", strings.NewReader("{}")))

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("peer received %d requests, want 1", got)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestIsConnectError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"dial", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, true},
		{"dns", &net.DNSError{Err: "no such host"}, true},
		{"read", &net.OpError{Op: "read", Err: errors.New("connection reset by peer")}, false},
		{"eof", io.EOF, false},
		{"wrapped dial", &url.Error{Op: "Post", Err: &net.OpError{Op: "dial", Err: errors.New("connection refused")}}, true},
		{"wrapped eof", &url.Error{Op: "Post", Err: io.EOF}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isConnectError(c.err); got != c.want {
				t.Fatalf("isConnectError(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

type failingResponseWriter struct {
	http.ResponseWriter
}

func (failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func newHandlerFixture(t *testing.T, resp interface{}, err error) (*RestHandler, *int32) {
	t.Helper()
	var calls int32
	rh := &RestHandler{}
	rh.AddListener("/default/api", func(interface{}) (interface{}, error) {
		atomic.AddInt32(&calls, 1)
		return resp, err
	})
	return rh, &calls
}

func TestHTTPHandlerReturnsNotFoundForUnmatchedPath(t *testing.T) {
	patches := patchCloudCoreIPs(t, localCloudCoreIP)
	defer patches.Reset()

	rec := httptest.NewRecorder()
	(&RestHandler{}).httpHandler(rec, httptest.NewRequest(http.MethodGet, "/node1/default/api", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec.Body.String() != "Request error" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "Request error")
	}
}

func TestHTTPHandlerWritesNothingForInvalidHandlerType(t *testing.T) {
	patches := patchCloudCoreIPs(t, localCloudCoreIP)
	defer patches.Reset()

	rh := &RestHandler{}
	rh.handlers.Store("/default/api", "not a handle")

	rec := httptest.NewRecorder()
	rh.httpHandler(rec, httptest.NewRequest(http.MethodGet, "/node1/default/api", nil))

	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
}

func TestHTTPHandlerWritesNothingWhenRuleHandlerFails(t *testing.T) {
	patches := patchCloudCoreIPs(t, localCloudCoreIP)
	defer patches.Reset()

	cases := []struct {
		name string
		resp interface{}
		err  error
	}{
		{"handler error", nil, errors.New("handle failed")},
		{"unexpected type", "not a response", nil},
		{"body read error", &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(failingReader{})}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rh, calls := newHandlerFixture(t, c.resp, c.err)
			rec := httptest.NewRecorder()
			rh.httpHandler(rec, httptest.NewRequest(http.MethodPost, "/node1/default/api", strings.NewReader("{}")))

			if got := atomic.LoadInt32(calls); got != 1 {
				t.Fatalf("handler invoked %d times, want 1", got)
			}
			if rec.Body.Len() != 0 {
				t.Fatalf("body = %q, want empty", rec.Body.String())
			}
		})
	}
}

func TestHTTPHandlerSurvivesClientWriteFailure(t *testing.T) {
	patches := patchCloudCoreIPs(t, localCloudCoreIP)
	defer patches.Reset()

	rh, calls := newHandlerFixture(t, &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("edge body")),
	}, nil)

	rec := httptest.NewRecorder()
	rh.httpHandler(failingResponseWriter{rec}, httptest.NewRequest(http.MethodPost, "/node1/default/api", strings.NewReader("{}")))

	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("handler invoked %d times, want 1", got)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHTTPHandlerSurvivesClientWriteFailureWhileForwarding(t *testing.T) {
	var calls int32
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = w.Write([]byte("peer body"))
	}))
	defer peer.Close()

	patches := patchCloudCoreIPs(t, "127.0.0.1")
	defer patches.Reset()

	rh := &RestHandler{port: serverPort(t, peer.URL)}
	rec := httptest.NewRecorder()
	rh.httpHandler(failingResponseWriter{rec}, httptest.NewRequest(http.MethodPost, "/node1/default/api", strings.NewReader("{}")))

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("peer received %d requests, want 1", got)
	}
}

func TestHTTPHandlerSurvivesClientWriteFailureForUnmatchedPath(t *testing.T) {
	patches := patchCloudCoreIPs(t, localCloudCoreIP)
	defer patches.Reset()

	rec := httptest.NewRecorder()
	(&RestHandler{}).httpHandler(failingResponseWriter{rec}, httptest.NewRequest(http.MethodGet, "/node1/default/api", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHTTPHandlerForwardsOverTLSWhenRequestIsTLS(t *testing.T) {
	var calls int32
	peer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = w.Write([]byte("peer body"))
	}))
	defer peer.Close()

	patches := patchCloudCoreIPs(t, "127.0.0.1")
	defer patches.Reset()

	rh := &RestHandler{port: serverPort(t, peer.URL)}
	req := httptest.NewRequest(http.MethodGet, "/node1/default/api", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	rh.httpHandler(rec, req)

	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("peer accepted %d requests, want 0 for an untrusted certificate", got)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHTTPHandlerReportsInvalidForwardRequest(t *testing.T) {
	patches := patchCloudCoreIPs(t, "127.0.0.1")
	defer patches.Reset()

	req := httptest.NewRequest(http.MethodGet, "/node1/default/api", nil)
	req.Method = "BAD METHOD"
	rec := httptest.NewRecorder()
	(&RestHandler{port: 9443}).httpHandler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rec.Body.String(), "failed to create forward request") {
		t.Fatalf("body = %q, want it to mention the forward request failure", rec.Body.String())
	}
}
