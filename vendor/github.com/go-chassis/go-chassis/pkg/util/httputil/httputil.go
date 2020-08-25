package httputil

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-mesh/openlogging"
	"io/ioutil"
	"net/http"
)

//ErrInvalidReq invalid input
var ErrInvalidReq = errors.New("rest consumer call arg is not *http.Request type")

//SetURI sets host for the request.
//set http(s)://{domain}/xxx
func SetURI(req *http.Request, url string) {
	if tempURL, err := req.URL.Parse(url); err == nil {
		req.URL = tempURL
	}
}

//SetBody is a method used for setting body for a request
func SetBody(req *http.Request, body []byte) {
	req.Body = ioutil.NopCloser(bytes.NewReader(body))
}

//SetCookie set key value in request cookie
func SetCookie(req *http.Request, k, v string) {
	c := &http.Cookie{
		Name:  k,
		Value: v,
	}
	req.AddCookie(c)
}

//GetCookie is a method which gets cookie from a request
func GetCookie(req *http.Request, key string) string {
	cookie, err := req.Cookie(key)
	if err == http.ErrNoCookie {
		return ""
	}
	return cookie.Value
}

// SetContentType is a method used for setting content-type in a request
func SetContentType(req *http.Request, ct string) {
	req.Header.Set("Content-Type", ct)
}

// GetContentType is a method used for getting content-type in a request
func GetContentType(req *http.Request) string {
	return req.Header.Get("Content-Type")
}

//HTTPRequest convert invocation to http request
func HTTPRequest(inv *invocation.Invocation) (*http.Request, error) {
	reqSend, ok := inv.Args.(*http.Request)
	if !ok {
		return nil, ErrInvalidReq
	}
	m := common.FromContext(inv.Ctx)
	for k, v := range m {
		reqSend.Header.Set(k, v)
	}
	return reqSend, nil
}

// ReadBody read body from the from the response
func ReadBody(resp *http.Response) []byte {
	if resp != nil && resp.Body != nil {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			openlogging.Error(fmt.Sprintf("read body failed: %s", err.Error()))
			return nil
		}
		return body
	}
	openlogging.Error("response body or response is nil")
	return nil
}

// GetRespCookie returns response Cookie.
func GetRespCookie(resp *http.Response, key string) []byte {
	for _, c := range resp.Cookies() {
		if c.Name == key {
			return []byte(c.Value)
		}
	}
	return nil
}

// SetRespCookie sets the cookie.
func SetRespCookie(resp *http.Response, cookie *http.Cookie) {
	resp.Header.Add("Set-Cookie", cookie.String())
}
