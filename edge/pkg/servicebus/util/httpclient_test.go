package util

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestURLClient_HTTPDo(t *testing.T) {
	client, err := GetURLClient(nil)
	if err != nil {
		t.Fatalf("GetURLClient error: %v", err)
	}

	ts := getMockServer(t)
	defer ts.Close()

	tests := []struct {
		method      string
		url         string
		headers     http.Header
		body        []byte
		expectError bool
		statusCode  int
	}{
		{"GET", ts.URL + "/test", nil, nil, false, http.StatusOK},
		{"POST", ts.URL + "/test", http.Header{"Content-Type": []string{"application/json"}}, []byte(`{"key":"value"}`), false, http.StatusOK},
		{"GET", ":", nil, nil, true, 0}, // Invalid URL
	}

	for _, tt := range tests {
		resp, err := client.HTTPDo(tt.method, tt.url, tt.headers, tt.body)
		if tt.expectError {
			if err == nil {
				t.Errorf("expected error for method %s with url %s, got none", tt.method, tt.url)
			}
			continue
		}
		if err != nil {
			t.Errorf("HTTPDo error: %v", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != tt.statusCode {
			t.Errorf("expected status code %d, got %d", tt.statusCode, resp.StatusCode)
		}
	}
}

func TestURLClient_SSL(t *testing.T) {
	client, err := GetURLClient(&URLClientOption{
		SSLEnabled:            true,
		TLSConfig:             &tls.Config{InsecureSkipVerify: true},
		ResponseHeaderTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("GetURLClient error: %v", err)
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := client.HTTPDo("GET", ts.URL+"/test", nil, nil)
	if err != nil {
		t.Errorf("HTTPDo error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestSignRequest(t *testing.T) {
	mockSignRequest := func(req *http.Request) error {
		req.Header.Set("Authorization", "Bearer token")
		return nil
	}
	defer func() { SignRequest = nil }()

	client, _ := GetURLClient(nil)
	ts := getMockServer(t)
	defer ts.Close()

	tests := []struct {
		signRequest func(*http.Request) error
		expectError bool
	}{
		{mockSignRequest, false},
		{func(req *http.Request) error { return errors.New("signing failed") }, true},
	}

	for _, tt := range tests {
		SignRequest = tt.signRequest
		resp, err := client.HTTPDo("GET", ts.URL+"/test", nil, nil)

		if tt.expectError && err == nil {
			t.Errorf("expected signing error but got none")
		} else if !tt.expectError && (err != nil || resp.StatusCode != http.StatusOK) {
			t.Errorf("unexpected signing failure: %v", err)
		}
	}
}

func getMockServer(_ *testing.T) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.EscapedPath() != "/test" {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		case http.MethodPost:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	return ts
}

func TestGetURLClientTimeoutDefaults(t *testing.T) {
	tests := map[string]struct {
		option             *URLClientOption
		wantHandshake      time.Duration
		wantResponseHeader time.Duration
	}{
		"nil option": {
			option:             nil,
			wantHandshake:      30 * time.Second,
			wantResponseHeader: 60 * time.Second,
		},
		"both unset": {
			option:             &URLClientOption{},
			wantHandshake:      30 * time.Second,
			wantResponseHeader: 60 * time.Second,
		},
		"only response header set": {
			option:             &URLClientOption{ResponseHeaderTimeout: 5 * time.Second},
			wantHandshake:      30 * time.Second,
			wantResponseHeader: 5 * time.Second,
		},
		"only handshake set": {
			option:             &URLClientOption{HandshakeTimeout: 5 * time.Second},
			wantHandshake:      5 * time.Second,
			wantResponseHeader: 60 * time.Second,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			client, err := GetURLClient(tt.option)
			if err != nil {
				t.Fatalf("GetURLClient error: %v", err)
			}
			if client.options.HandshakeTimeout != tt.wantHandshake {
				t.Errorf("HandshakeTimeout: got %v, want %v",
					client.options.HandshakeTimeout, tt.wantHandshake)
			}
			if client.options.ResponseHeaderTimeout != tt.wantResponseHeader {
				t.Errorf("ResponseHeaderTimeout: got %v, want %v",
					client.options.ResponseHeaderTimeout, tt.wantResponseHeader)
			}
		})
	}
}
