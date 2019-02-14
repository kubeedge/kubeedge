/*
Copyright 2019 The KubeEdge Authors.

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

package http

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
	common "github.com/kubeedge/kubeedge/edge/pkg/edgehub/common"
)

const (
	CesMetricDataURL = "https://huaweicloud.com/V1.0"
	CertFile         = "/tmp/kubeedge/testData/edge.crt"
	KeyFile          = "/tmp/kubeedge/testData/edge.key"
)

//TestBuildRequest() tests the building of a proper http Requests
func TestBuildRequest(t *testing.T) {
	_, err := BuildRequest(http.MethodPost, CesMetricDataURL, nil, "")
	if err != nil {
		t.Fatalf("Failed to build http request , error is %v", err.Error())
	}
}

//TestBuildRequestInvalidMethod() tests the validation of incorrect HTTP methods
func TestBuildRequestInvalidMethod(t *testing.T) {
	_, err := BuildRequest("#$%@!@", CesMetricDataURL, nil, "")
	if err == nil {
		t.Fatalf("Failed to validate incorrect HTTP method")
	}
}

//TestBuildRequestWithToken() tests the building of HTTP requests with token
func TestBuildRequestWithToken(t *testing.T) {
	req, err := BuildRequest(http.MethodPost, CesMetricDataURL, nil, "abc")
	if err != nil {
		t.Fatalf("Failed to build request with token , error is %v", err.Error())
	}
	assert.NotNil(t, req.Header.Get("Content-Type"))
}

//TestBuildRequestWithBody() tests the building of HTTP requests with a body present
func TestBuildRequestWithBody(t *testing.T) {
	body, _ := ioutil.ReadFile("request.json")
	ioBody := bytes.NewReader(body)
	_, err := BuildRequest(http.MethodPost, CesMetricDataURL, ioBody, "abc")
	if err != nil {
		t.Fatalf("Failed to build request with token , error is %v", err.Error())
	}
}

//TestNewHttpClient() tests the creation of a new HTTP client
func TestNewHttpClient(t *testing.T) {
	httpClient := NewHTTPClient()
	if httpClient == nil {
		t.Fatal("Failed to build HTTP client")
	}
}

//TestSendRequest() tests the send functionality of HTTP requests
func TestSendRequest(t *testing.T) {
	body, _ := ioutil.ReadFile("request.json")
	ioBody := bytes.NewReader(body)
	req, err := BuildRequest(http.MethodPost, CesMetricDataURL, ioBody, "abc")
	if err != nil {
		t.Fatalf("Failed to build request with token , error is %v", err.Error())
	}
	resp, err := SendRequest(req, NewHTTPClient())
	assert.NotNil(t, resp)
	common.AssertIntEqual(t, strconv.Itoa(http.StatusNotFound), strconv.Itoa(resp.StatusCode), "return code not match")

}

//TestNewHTTPSclient() tests the creation of a new HTTPS client with proper values
func TestNewHTTPSclient(t *testing.T) {
	err := util.GenerateTestCertificate("/tmp/kubeedge/testData/", "edge", "edge")
	if err != nil {
		t.Errorf("Error in generating fake certificates: %v", err)
		return
	}
	certificate, err := tls.LoadX509KeyPair(CertFile, KeyFile)
	if err != nil {
		t.Errorf("Error in loading key pair: %v", err)
		return
	}
	type args struct {
		certFile string
		keyFile  string
	}
	tests := []struct {
		name    string
		args    args
		want    *http.Client
		wantErr bool
	}{
		{"TestNewHTTPSclient: ", args{
			keyFile:  KeyFile,
			certFile: CertFile,
		}, &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      x509.NewCertPool(),
					Certificates: []tls.Certificate{certificate},
					MinVersion:   tls.VersionTLS12,
					CipherSuites: []uint16{
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					},
					InsecureSkipVerify: true},
			},
			Timeout: connectTimeout,
		}, false},

		{"Wrong path given while getting HTTPS client", args{
			keyFile:  "WrongKeyFilePath",
			certFile: "WrongCertFilePath",
		}, nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewHTTPSclient(tt.args.certFile, tt.args.keyFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHTTPSclient() error = %v, expectedError = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHTTPSclient() = %v, want %v", got, tt.want)
			}
		})
	}
}
