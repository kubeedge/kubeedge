package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestURLClient_HTTPDo(t *testing.T) {
	client, err := GetURLClient(nil)
	if err != nil {
		t.Errorf("GetURLClient error: %v", err)
	}

	ts := getMockServer(t)
	resp, err := client.HTTPDo("GET", ts.URL+"/test", http.Header{}, nil)
	if err != nil {
		t.Errorf("HTTPDo error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got error status code, resp is %v", resp)
	}
}

func getMockServer(t *testing.T) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			if r.URL.EscapedPath() != "/test" {
				t.Errorf("path error: %s", r.URL.EscapedPath())
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}
	}))

	return ts
}
