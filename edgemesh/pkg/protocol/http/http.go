package http

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edgemesh/pkg/config"
)

type HTTP struct {
	Conn         net.Conn
	SvcNamespace string
	SvcName      string
	Port         int

	req *http.Request
}

// Process handles http protocol
func (p *HTTP) Process() {
	defer p.Conn.Close()

	for {
		// parse http request
		req, err := http.ReadRequest(bufio.NewReader(p.Conn))
		if err != nil {
			if err == io.EOF {
				klog.Infof("[EdgeMesh] http client disconnected.")
				return
			}
			klog.Errorf("[EdgeMesh] parse http request err: %v", err)
			return
		}

		// http: Request.RequestURI can't be set in client requests
		// just reset it before transport
		req.RequestURI = ""

		// create invocation
		inv := invocation.New(context.Background())

		// set invocation
		inv.MicroServiceName = req.Host
		inv.SourceServiceID = ""
		inv.Protocol = "rest"
		inv.Strategy = config.Config.LBStrategy
		inv.Args = req
		inv.Reply = &http.Response{}

		// create handlerchain
		c, err := handler.CreateChain(common.Consumer, "http", handler.Loadbalance, handler.Transport)
		if err != nil {
			klog.Errorf("[EdgeMesh] create http handlerchain error: %v", err)
			return
		}

		// start to handle
		p.req = req
		c.Next(inv, p.responseCallback)
	}
}

// responseCallback implements http handlerchain callback
func (p *HTTP) responseCallback(data *invocation.Response) error {
	var err error
	var resp *http.Response
	var respBytes []byte

	// as a proxy server, make sure that edgemesh always response to a request
	// send either the response of the real backend server or 503 back
	if data.Err != nil {
		klog.Errorf("[EdgeMesh] error in http handlerchain: %v", data.Err)
		err = data.Err
	} else {
		if data.Result == nil {
			klog.Errorf("[EdgeMesh] empty response from http handlerchain")
			err = fmt.Errorf("empty response from http handlerchain")
		} else {
			var ok bool
			resp, ok = data.Result.(*http.Response)
			if !ok {
				klog.Errorf("[EdgeMesh] http handlerchain result %+v not *http.Response type", data.Result)
				err = fmt.Errorf("result not *http.Response type")
			} else {
				respBytes, err = httpResponseToBytes(resp)
				if err != nil {
					klog.Errorf("[EdgeMesh] convert http response to bytes err: %v", err)
				} else {
					// send response back
					if _, err := p.Conn.Write(respBytes); err != nil {
						klog.Errorf("[EdgeMesh] write err: %v", err)
					}
					return nil
				}
			}
		}
	}
	// 503
	resp = &http.Response{
		Status:     fmt.Sprintf("%d %s", http.StatusServiceUnavailable, err),
		StatusCode: http.StatusServiceUnavailable,
		Proto:      p.req.Proto,
		Request:    p.req,
		Header:     make(http.Header),
	}
	respBytes, _ = httpResponseToBytes(resp)
	// send error response back
	if _, err = p.Conn.Write(respBytes); err != nil {
		return err
	}
	return nil
}

// httpResponseToBytes transforms http.Response to bytes
func httpResponseToBytes(resp *http.Response) ([]byte, error) {
	buf := new(bytes.Buffer)
	if resp == nil {
		return nil, fmt.Errorf("http response nil")
	}
	err := resp.Write(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
