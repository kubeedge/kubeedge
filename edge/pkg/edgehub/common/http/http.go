package http

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
)

const (
	defaultConnectTimeout            = 30 * time.Second
	defaultKeepAliveTimeout          = 30 * time.Second
	defaultResponseReadTimeout       = 300 * time.Second
	defaultMaxIdleConnectionsPerHost = 3
)

var (
	connectTimeout            = defaultConnectTimeout
	keepaliveTimeout          = defaultKeepAliveTimeout
	responseReadTimeout       = defaultResponseReadTimeout
	maxIdleConnectionsPerHost = defaultMaxIdleConnectionsPerHost
)

// NewHTTPClient create new client
func NewHTTPClient() *http.Client {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   connectTimeout,
			KeepAlive: keepaliveTimeout,
		}).Dial,
		MaxIdleConnsPerHost:   maxIdleConnectionsPerHost,
		ResponseHeaderTimeout: responseReadTimeout,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	log.LOGGER.Infof("tlsConfig InsecureSkipVerify true")
	return &http.Client{Transport: transport}
}

// NewHTTPSclient create https client
func NewHTTPSclient(certFile, keyFile string) (*http.Client, error) {
	pool := x509.NewCertPool()
	cliCrt, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.LOGGER.Errorf("Cannot create https client , Loadx509keypair err: %v", err)
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:      pool,
			Certificates: []tls.Certificate{cliCrt},
			MinVersion:   tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
			InsecureSkipVerify: true}, /*Now we need set it true*/
	}
	client := &http.Client{Transport: tr, Timeout: connectTimeout}
	return client, nil
}

// SendRequest sends a http request and return the resp info
func SendRequest(req *http.Request, client *http.Client) (*http.Response, error) {
	//body, err := httputil.DumpRequest(req, true)
	//if err != nil {
	//	return nil, err
	//}
	//log.LOGGER.Debugf("POST request : %s", string(body))
	//log.LOGGER.Debugf("url: %#v", req.URL)
	//log.LOGGER.Debugf("header: %#v", req.Header)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// BuildRequest Creates a HTTP request.
func BuildRequest(method string, urlStr string, body io.Reader, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Add("X-Auth-Token", token)
	}
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}
