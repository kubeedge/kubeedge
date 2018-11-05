package http

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	common "kubeedge/pkg/edgehub/common"
	"github.com/stretchr/testify/assert"
)

const (
	CesMetricDataURL = "https://ces.kwe.devcloud.huawei.com/V1.0"
	// CA_FILE             = "/tmp/ca.crt"
	// CERT_FILE           = "/tmp/client.cert"
	// KEY_FILE            = "/tmp/client.key"
)

func TestBuildRequest(t *testing.T) {
	_, err := BuildRequest(http.MethodPost, CesMetricDataURL, nil, "")
	if err != nil {
		t.Fatalf("Failed to build http request , error is %v", err.Error())
	}
}

func TestBuildRequestInvalidMethod(t *testing.T) {

	_, err := BuildRequest("#$%@!@", CesMetricDataURL, nil, "")
	if err == nil {
		t.Fatalf("Failed to validate incorrect HTTP method")
	}
}

func TestBuildRequestWithToken(t *testing.T) {
	req, err := BuildRequest(http.MethodPost, CesMetricDataURL, nil, "abc")
	if err != nil {
		t.Fatalf("Failed to build request with token , error is %v", err.Error())
	}
	assert.NotNil(t, req.Header.Get("Content-Type"))
}

func TestBuildRequestWithBody(t *testing.T) {
	body, _ := ioutil.ReadFile("request.json")
	ioBody := bytes.NewReader(body)
	_, err := BuildRequest(http.MethodPost, CesMetricDataURL, ioBody, "abc")
	if err != nil {
		t.Fatalf("Failed to build request with token , error is %v", err.Error())
	}
}

func TestNewHttpClient(t *testing.T) {
	httpClient := NewHTTPClient()
	if httpClient == nil {
		t.Fatal("Failed to build HTTP client")
	}
}

// func TestBuildHttpsClient(t *testing.T) {
// 	config := edgemonitor.Config{}
// 	os.Create(CA_FILE)
// 	os.Create(CERT_FILE)
// 	os.Create(KEY_FILE)
// 	config.CaFile = CA_FILE
// 	config.CertFile = CERT_FILE
// 	config.KeyFile = KEY_FILE
// 	client, _ := NewHTTPSclient(&config)
// 	assert.Nil(t, client)
// 	os.Remove(CA_FILE)
// 	os.Remove(CERT_FILE)
// 	os.Remove(KEY_FILE)
// }

// func TestBuildHttpsClientInvalidCertPath(t *testing.T) {
// 	config := edgemonitor.Config{}
// 	config.CaFile = "abc"
// 	config.CertFile = "abc"
// 	config.KeyFile = "abc"
// 	client, err := NewHTTPSclient(&config)
// 	if err == nil {
// 		t.Fatal("Failed to validate invalid certificate path")
// 	}
// 	assert.Nil(t, client)
// }

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
