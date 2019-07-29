package handler

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/control"
	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/pkg/circuit"
	"github.com/go-chassis/go-chassis/third_party/forked/afex/hystrix-go/hystrix"
)

// constant for bizkeeper-consumer
const (
	Name = "bizkeeper-consumer"
)

// BizKeeperConsumerHandler bizkeeper consumer handler
type BizKeeperConsumerHandler struct{}

// Handle function is for to handle the chain
func (bk *BizKeeperConsumerHandler) Handle(chain *Chain, i *invocation.Invocation, cb invocation.ResponseCallBack) {
	command, cmdConfig := control.DefaultPanel.GetCircuitBreaker(*i, common.Consumer)

	cmdConfig.MetricsConsumerNum = archaius.GetInt("cse.metrics.circuitMetricsConsumerNum", hystrix.DefaultMetricsConsumerNum)
	hystrix.ConfigureCommand(command, cmdConfig)

	finish := make(chan *invocation.Response, 1)
	f, err := GetFallbackFun(command, common.Consumer, i, finish, cmdConfig.ForceFallback)
	if err != nil {
		writeErr(err, cb)
		return
	}
	err = hystrix.Do(command, func() (err error) {
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
	}, f)

	//if err is not nil, means fallback is nil, return original err
	if err != nil {
		writeErr(err, cb)
		return
	}

	cb(<-finish)
}

// GetFallbackFun get fallback function
func GetFallbackFun(cmd, t string, i *invocation.Invocation, finish chan *invocation.Response, isForce bool) (func(error) error, error) {
	enabled := config.GetFallbackEnabled(cmd, t)
	if enabled || isForce {
		p := config.GetPolicy(i.MicroServiceName, t)
		if p == "" {
			p = circuit.ReturnErr
		}
		f, err := circuit.GetFallback(p)
		if err != nil {
			return nil, err
		}
		return f(i, finish), nil
	}
	return nil, nil
}

// newBizKeeperConsumerHandler new bizkeeper consumer handler
func newBizKeeperConsumerHandler() Handler {
	return &BizKeeperConsumerHandler{}
}

// Name is for to represent the name of bizkeeper handler
func (bk *BizKeeperConsumerHandler) Name() string {
	return Name
}
