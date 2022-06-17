package util

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestURLClient_HTTPDo(t *testing.T) {

	type args struct {
		option *URLClientOption
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestURLClient_HTTPDo(): Case 1: SuccessCase",
			args: args{
				&URLClientOption{
					SSLEnabled:            true,
					Compressed:            true,
					HandshakeTimeout:      0,
					ResponseHeaderTimeout: 0,
					TLSConfig:             &tls.Config{InsecureSkipVerify: true},
				},
			},
			wantErr: false,
		},
		{
			name: "TestURLClient_HTTPDo(): Case 2: SuccessCase",
			args: args{
				&URLClientOption{
					SSLEnabled:            false,
					Compressed:            true,
					HandshakeTimeout:      0,
					ResponseHeaderTimeout: 0,
					TLSConfig:             nil,
				},
			},
			wantErr: false,
		},
		{
			name: "TestURLClient_HTTPDo(): Case 3: SuccessCase",
			args: args{
				nil,
			},
			wantErr: false,
		}, {
			name: "TestURLClient_HTTPDo(): Case 4: FailureCase",
			args: args{
				&URLClientOption{
					SSLEnabled:            true,
					Compressed:            true,
					HandshakeTimeout:      0,
					ResponseHeaderTimeout: 0,
					TLSConfig:             nil,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := GetURLClient(tt.args.option)
			if err != nil {
				t.Errorf("GetURLClient() error: %v", err)
			}
			ts := getMockHttpServer(t, tt.args.option)
			resp, err := client.HTTPDo("GET", ts.URL+"/test", nil, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPDo() error = %v, expectedError = %v", err, tt.wantErr)
			} else if err == nil {
				defer resp.Body.Close()
			}
		})
	}
}

func getMockHttpServer(t *testing.T, option *URLClientOption) *httptest.Server {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			if r.URL.EscapedPath() != "/test" {
				t.Errorf("Path error: %s", r.URL.EscapedPath())
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}
	}))
	if option != nil && option.SSLEnabled {
		ts.StartTLS()
	} else {
		ts.Start()
	}
	return ts
}
