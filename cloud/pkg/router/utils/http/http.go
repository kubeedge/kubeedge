package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"k8s.io/klog/v2"
)

const (
	defaultConnectTimeout            = 30 * time.Second
	defaultKeepAliveTimeout          = 30 * time.Second
	defaultResponseReadTimeout       = 300 * time.Second
	defaultIdleConnTimeout           = 90 * time.Second
	defaultMaxIdleConnectionsPerHost = 3
)

var (
	connectTimeout            = defaultConnectTimeout
	keepaliveTimeout          = defaultKeepAliveTimeout
	responseReadTimeout       = defaultResponseReadTimeout
	idleConnTimeout           = defaultIdleConnTimeout
	maxIdleConnectionsPerHost = defaultMaxIdleConnectionsPerHost
)

// NewHTTPClient create new client
func NewHTTPClient() *http.Client {
	klog.Infof("tlsConfig InsecureSkipVerify true")
	return NewHTTPClientWithTLSConfig(&tls.Config{InsecureSkipVerify: true})
}

// NewHTTPClientWithTLSConfig creates a client that uses tlsConfig for https requests.
// The client keeps a connection pool, so callers should create it once and reuse it.
func NewHTTPClientWithTLSConfig(tlsConfig *tls.Config) *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   connectTimeout,
			KeepAlive: keepaliveTimeout,
		}).DialContext,
		MaxIdleConnsPerHost:   maxIdleConnectionsPerHost,
		IdleConnTimeout:       idleConnTimeout,
		ResponseHeaderTimeout: responseReadTimeout,
		TLSClientConfig:       tlsConfig,
	}
	return &http.Client{Transport: transport}
}

// NewRestTargetTLSConfig creates the TLS config used to call https REST targets.
// If insecureSkipVerify is false, target certificates are verified against the
// system roots plus the certificates in caFile, if caFile is set.
func NewRestTargetTLSConfig(caFile string, insecureSkipVerify bool) (*tls.Config, error) {
	if insecureSkipVerify {
		return &tls.Config{InsecureSkipVerify: true}, nil
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		klog.Warningf("failed to load system cert pool, only the certificates in caFile are trusted: %v", err)
		pool = x509.NewCertPool()
	}
	if caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read caFile %s: %v", caFile, err)
		}
		if ok := pool.AppendCertsFromPEM(caPEM); !ok {
			return nil, fmt.Errorf("no certificates found in caFile %s", caFile)
		}
	}
	return &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}, nil
}

// NewHTTPSClient create https client
func NewHTTPSClient(certFile, keyFile string) (*http.Client, error) {
	pool := x509.NewCertPool()
	cliCrt, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		klog.Errorf("Cannot create https client , Load x509 key pair err: %v", err)
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

// NewHTTPClientWithCA create client without certificate
func NewHTTPClientWithCA(capem []byte, certificate tls.Certificate) (*http.Client, error) {
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(capem); !ok {
		return nil, fmt.Errorf("cannot parse the certificates")
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: false,
			Certificates:       []tls.Certificate{certificate},
		},
	}
	client := &http.Client{Transport: tr, Timeout: connectTimeout}
	return client, nil
}

// SendRequest sends a http request and return the resp info
func SendRequest(req *http.Request, client *http.Client) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// BuildRequest Creates a HTTP request.
func BuildRequest(method string, urlStr string, body io.Reader, token string, nodeName string) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	if token != "" {
		bearerToken := "Bearer " + token
		req.Header.Add("Authorization", bearerToken)
	}
	if nodeName != "" {
		req.Header.Add("NodeName", nodeName)
	}
	return req, nil
}
