package util

import (
	"bytes"
	"crypto/tls"
	"errors"
	"net/http"
	"strings"
	"time"
)

// SignRequest sign a http request so that it can talk to API Server
var SignRequest func(*http.Request) error

// URLClientOption is a struct which provides options for client
type URLClientOption struct {
	SSLEnabled            bool
	TLSConfig             *tls.Config
	Compressed            bool
	HandshakeTimeout      time.Duration
	ResponseHeaderTimeout time.Duration
	Verbose               bool
}

// URLClient is a struct used for storing details of a client
type URLClient struct {
	*http.Client
	TLS     *tls.Config
	Request *http.Request
	options URLClientOption
}

// HTTPDo is a method used for http connection
func (client *URLClient) HTTPDo(method, rawURL string, headers http.Header, body []byte) (resp *http.Response, err error) {
	client.clientHasPrefix(rawURL, "https")

	if headers == nil {
		headers = make(http.Header)
	}

	if _, ok := headers["Accept"]; !ok {
		headers["Accept"] = []string{"*/*"}
	}
	if _, ok := headers["Accept-Encoding"]; !ok && client.options.Compressed {
		headers["Accept-Encoding"] = []string{"deflate, gzip"}
	}

	req, err := http.NewRequest(method, rawURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	client.Request = req

	req.Header = headers
	//sign a request
	if SignRequest != nil {
		if err = SignRequest(req); err != nil {
			return nil, errors.New("Add auth info failed, err: " + err.Error())
		}
	}
	resp, err = client.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (client *URLClient) clientHasPrefix(url, pro string) {
	if strings.HasPrefix(url, pro) {
		if transport, ok := client.Client.Transport.(*http.Transport); ok {
			transport.TLSClientConfig = client.TLS
		}
	}
}

// DefaultURLClientOption is a struct object which has default client option
var DefaultURLClientOption = &URLClientOption{
	Compressed:            true,
	HandshakeTimeout:      30 * time.Second,
	ResponseHeaderTimeout: 60 * time.Second,
}

// GetURLClient is a function which sets client options
func GetURLClient(option *URLClientOption) (client *URLClient, err error) {
	if option == nil {
		option = DefaultURLClientOption
	} else {
		switch {
		case option.HandshakeTimeout == 0:
			option.HandshakeTimeout = DefaultURLClientOption.HandshakeTimeout
			fallthrough
		case option.ResponseHeaderTimeout == 0:
			option.ResponseHeaderTimeout = DefaultURLClientOption.HandshakeTimeout
		}
	}

	if !option.SSLEnabled {
		client = &URLClient{
			Client: &http.Client{
				Transport: &http.Transport{
					TLSHandshakeTimeout:   option.HandshakeTimeout,
					ResponseHeaderTimeout: option.ResponseHeaderTimeout,
					DisableCompression:    !option.Compressed,
				},
			},
			options: *option,
		}

		return
	}

	client = &URLClient{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSHandshakeTimeout:   option.HandshakeTimeout,
				ResponseHeaderTimeout: option.ResponseHeaderTimeout,
				DisableCompression:    !option.Compressed,
			},
		},
		TLS:     option.TLSConfig,
		options: *option,
	}
	return
}
