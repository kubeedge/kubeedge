package handler

import (
	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/invocation"

	"github.com/go-chassis/go-chassis/control"
	"github.com/go-chassis/go-chassis/third_party/forked/afex/hystrix-go/hystrix"
)

// BizKeeperProviderHandler bizkeeper provider handler
type BizKeeperProviderHandler struct{}

// Handle handler for bizkeeper provider
func (bk *BizKeeperProviderHandler) Handle(chain *Chain, i *invocation.Invocation, cb invocation.ResponseCallBack) {
	command, cmdConfig := control.DefaultPanel.GetCircuitBreaker(*i, common.Provider)
	hystrix.ConfigureCommand(command, cmdConfig)

	finish := make(chan *invocation.Response, 1)
	err := hystrix.Do(command, func() (err error) {
		chain.Next(i, func(resp *invocation.Response) error {
			err = resp.Err
			select {
			case finish <- resp:
			default:
				// means hystrix error occurred
			}
			return err
		})
		return
	}, nil)

	//if err is not nil, means fallback is nil, return original err
	if err != nil {
		writeErr(err, cb)
	}

	cb(<-finish)
}

// Name returns bizkeeper-provider string
func (bk *BizKeeperProviderHandler) Name() string {
	return "bizkeeper-provider"
}

func newBizKeeperProviderHandler() Handler {
	return &BizKeeperProviderHandler{}
}
