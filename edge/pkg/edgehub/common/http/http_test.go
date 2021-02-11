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
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
)

const (
	CertFile = "/tmp/kubeedge/testData/edge.crt"
	KeyFile  = "/tmp/kubeedge/testData/edge.key"
	Method   = "GET"
	Url      = "kubeedge.io"
)

//TestNewHttpClient() tests the creation of a new HTTP client
func TestNewHttpClient(t *testing.T) {
	httpClient := NewHTTPClient()
	if httpClient == nil {
		t.Fatal("Failed to build HTTP client")
	}
}

//TestNewHTTPSClient() tests the creation of a new HTTPS client with proper values
func TestNewHTTPSClient(t *testing.T) {
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
		{
			name: "TestNewHTTPSClient: ",
			args: args{
				keyFile:  KeyFile,
				certFile: CertFile,
			},
			want: &http.Client{
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
			},
			wantErr: false,
		},
		{
			name: "Wrong path given while getting HTTPS client",
			args: args{
				keyFile:  "WrongKeyFilePath",
				certFile: "WrongCertFilePath",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewHTTPSClient(tt.args.certFile, tt.args.keyFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHTTPSClient() error = %v, expectedError = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHTTPSClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

//TestNewHTTPClientWithCA() tests the creation of a new HTTP using filled capem
func TestNewHTTPClientWithCA(t *testing.T) {
	err := util.GenerateTestCertificate("/tmp/kubeedge/testData/", "edge", "edge")
	if err != nil {
		t.Errorf("Error in generating fake certificates: %v", err)
		return
	}
	capem, err := ioutil.ReadFile(CertFile)
	if err != nil {
		t.Errorf("Error in loading Cert file: %v", err)
		return
	}
	certificate := tls.Certificate{}

	testPool := x509.NewCertPool()
	if ok := testPool.AppendCertsFromPEM(capem); !ok {
		t.Errorf("cannot parse the certificates")
		return
	}

	type args struct {
		capem       []byte
		certificate tls.Certificate
	}
	tests := []struct {
		name    string
		args    args
		want    *http.Client
		wantErr bool
	}{
		{
			name: "TestNewHTTPClientWithCA: ",
			args: args{
				capem:       capem,
				certificate: certificate,
			},
			want: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs:            testPool,
						InsecureSkipVerify: false,
						Certificates:       []tls.Certificate{certificate},
					},
				},
				Timeout: connectTimeout,
			},
			wantErr: false,
		},
		{
			name: "Wrong certifcate given when getting HTTP client",
			args: args{
				capem:       []byte{},
				certificate: certificate,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewHTTPClientWithCA(tt.args.capem, tt.args.certificate)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHTTPClientWithCA() error = %v, expectedError = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHTTPClientWithCA() = %v, want %v", got, tt.want)
			}
		})
	}
}

//TestBuildRequest() tests the process of message building
func TestBuildRequest(t *testing.T) {
	reader := bytes.NewReader([]byte{})
	token := "token"
	nodeName := "name"

	req, err := http.NewRequest(Method, Url, reader)
	if err != nil {
		t.Errorf("Error in creating new http request message: %v", err)
		return
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("NodeName", nodeName)

	type args struct {
		method   string
		urlStr   string
		body     io.Reader
		token    string
		nodeName string
	}
	tests := []struct {
		name    string
		args    args
		want    *http.Request
		wantErr bool
	}{
		{
			name: "TestBuildRequest: ",
			args: args{
				method:   Method,
				urlStr:   Url,
				body:     reader,
				token:    token,
				nodeName: nodeName,
			},
			want:    req,
			wantErr: false,
		},
		{
			name: "NewRequest failure causes BuildRequest failure: ",
			args: args{
				method:   "INVALID\n",
				urlStr:   Url,
				body:     reader,
				token:    token,
				nodeName: nodeName,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildRequest(tt.args.method, tt.args.urlStr, tt.args.body, tt.args.token, tt.args.nodeName)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildRequest() error = %v, expectedError = %v", err, tt.wantErr)
				return
			}
			//needed to handle failure testcase because can't deep compare field in nil
			if got == tt.want && err != nil && tt.wantErr == true {
				return
			}
			if !reflect.DeepEqual(got.Header, tt.want.Header) {
				t.Errorf("BuildRequest() Header = %v, want %v", got, tt.want.Header)
			}
			if !reflect.DeepEqual(got.Body, tt.want.Body) {
				t.Errorf("BuildRequest() Body = %v, want %v", got, tt.want.Body)
			}
		})
	}
}

//TestSendRequestFailure() uses fake data and expects function to fail
func TestSendRequestFailure(t *testing.T) {
	httpClient := NewHTTPClient()
	if httpClient == nil {
		t.Fatal("Failed to build HTTP client")
	}

	req, err := http.NewRequest(Method, Url, bytes.NewReader([]byte{}))
	if err != nil {
		t.Errorf("Error in creating new http request message: %v", err)
		return
	}

	resp, respErr := SendRequest(req, httpClient)
	if resp != nil && respErr == nil {
		t.Errorf("Error, response should not come as data is not valid")
		return
	}
}
