package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chassis/go-chassis/client/rest"
	"github.com/go-chassis/go-chassis/core/client"
	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/pkg/util/httputil"
	"net/http"
)

// Test is the function to call provider health check api and check the response
func Test(ctx context.Context, protocol, endpoint string, expected Reply) (err error) {
	switch protocol {
	case common.ProtocolRest:
		err = restTest(ctx, endpoint, expected)
	default:
		err = fmt.Errorf("unsupport protocol %s", protocol)
	}
	return
}

func restTest(ctx context.Context, endpoint string, expected Reply) (err error) {
	c, err := client.GetClient(common.ProtocolRest, expected.ServiceName, "")
	if err != nil {
		return
	}

	arg, _ := rest.NewRequest(http.MethodGet, "http://"+expected.ServiceName+"/healthz", nil)
	req := &invocation.Invocation{Args: arg}
	rsp := rest.NewResponse()
	err = c.Call(ctx, endpoint, req, rsp)
	if rsp.Body != nil {
		defer rsp.Body.Close()
	}
	if err != nil {
		return
	}
	if rsp.StatusCode != http.StatusOK {
		return nil
	}
	var actual Reply
	err = json.Unmarshal(httputil.ReadBody(rsp), &actual)
	if err != nil {
		return
	}
	if actual != expected {
		return fmt.Errorf("endpoint is belong to %s:%s:%s",
			actual.ServiceName, actual.Version, actual.AppID)
	}
	return
}
