package listener

import (
	"context"
	"io"
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
		w.Header().Set("X-Peer", "value")
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte("peer body"))
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
