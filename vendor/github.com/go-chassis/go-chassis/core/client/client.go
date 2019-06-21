// Package client is an interface for any protocol's client
package client

import (
	"context"
	"errors"

	"github.com/go-chassis/go-chassis/core/invocation"
)

//ErrCanceled means Request is canceled by context management
var ErrCanceled = errors.New("request cancelled")

//TransportFailure is caused by client call failure
//for example:  resp, err = client.Do(req)
//if err is not nil then should wrap original error with TransportFailure
type TransportFailure struct {
	Message string
}

// Error return error message
func (e TransportFailure) Error() string {
	return e.Message
}

// ProtocolClient is the interface to communicate with one kind of ProtocolServer, it is used in transport handler
// rcp protocol client,http protocol client,or you can implement your own
type ProtocolClient interface {
	// TODO use invocation.Response as rsp
	Call(ctx context.Context, addr string, inv *invocation.Invocation, rsp interface{}) error
	String() string
	Close() error
	ReloadConfigs(Options)
	GetOptions() Options
}
