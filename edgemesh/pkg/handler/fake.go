package handler

import (
	"net/http"
	"net/url"

	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/kubeedge/beehive/pkg/common/log"
)

//currently all handlers we use are already implemented by go-chassis, later we can add our own handlers here

type HTTPTestHandler struct{}

//Handle
func (h *HTTPTestHandler) Handle(chain *handler.Chain, inv *invocation.Invocation, cb invocation.ResponseCallBack) {
	r := &invocation.Response{
		Err: nil,
	}
	req := inv.Args.(*http.Request)
	clt := &http.Client{}
	u, err := url.Parse("http://127.0.0.1:8888")
	if err != nil {
		log.LOGGER.Errorf("Parse new url error: %v", err)
		r.Err = err
	} else {
		req.URL = u
		resp, err := clt.Do(req)
		if err != nil {
			log.LOGGER.Errorf("Transfer request to test HTTP server error: %v", err)
			r.Err = err
		} else {
			r.Result = resp
		}
	}
	cb(r)
}

//Name
func (h *HTTPTestHandler) Name() string {
	return "httpTestHandler"
}
func NewHTTPTestHandler() handler.Handler {
	return &HTTPTestHandler{}
}
