package handler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-mesh/openlogging"
)

var errEmptyChain = errors.New("chain can not be empty")

// ChainMap just concurrent read
var ChainMap = make(map[string]*Chain)

// Chain struct for service and handlers
type Chain struct {
	ServiceType  string
	Name         string
	Handlers     []Handler
	HandlerIndex int
}

// AddHandler chain can add a handler
func (c *Chain) AddHandler(h Handler) {
	c.Handlers = append(c.Handlers, h)
}

// Next is for to handle next handler in the chain
func (c *Chain) Next(i *invocation.Invocation, f invocation.ResponseCallBack) {
	index := c.HandlerIndex
	if index >= len(c.Handlers) {
		r := &invocation.Response{
			Err: nil,
		}
		f(r)
		return
	}
	c.HandlerIndex++
	c.Handlers[index].Handle(c, i, f)
}

// Reset for to reset the handler index
func (c *Chain) Reset() {
	c.HandlerIndex = 0
}

// ChainOptions chain options
type ChainOptions struct {
	Name string
}

// ChainOption is a function name
type ChainOption func(*ChainOptions)

// WithChainName returns the name of the chain option
func WithChainName(name string) ChainOption {
	return func(c *ChainOptions) {
		c.Name = name
	}
}

// parseHandlers for parsing the handlers
func parseHandlers(handlerStr string) []string {
	formatNames := strings.Replace(strings.TrimSpace(handlerStr), " ", "", -1)
	handlerNames := strings.Split(formatNames, ",")
	var s []string
	//delete empty string
	for _, v := range handlerNames {
		if v != "" {
			s = append(s, v)
		}
	}
	return s
}

//CreateChains create the chains based on type and handler map
func CreateChains(chainType string, handlerNameMap map[string]string) error {
	for chainName := range handlerNameMap {
		handlerNames := parseHandlers(handlerNameMap[chainName])
		c, err := CreateChain(chainType, chainName, handlerNames...)
		if err != nil {
			return fmt.Errorf("err create chain %s.%s:%s %s", chainType, chainName, handlerNames, err.Error())
		}
		ChainMap[chainType+chainName] = c

	}
	return nil
}

//CreateChain create consumer or provider's chain,the handlers is different
func CreateChain(serviceType string, chainName string, handlerNames ...string) (*Chain, error) {
	c := &Chain{
		ServiceType: serviceType,
		Name:        chainName,
	}
	openlogging.Debug(fmt.Sprintf("add [%d] handlers for chain [%s]", len(handlerNames), chainName))

	for _, name := range handlerNames {
		err := addHandler(c, name)
		if err != nil {
			return nil, err
		}
	}

	if len(c.Handlers) == 0 {
		openlogging.Warn("Chain " + chainName + " is Empty")
		return c, nil
	}
	return c, nil
}

// addHandler add handler
func addHandler(c *Chain, name string) error {
	handler, err := CreateHandler(name)
	if err != nil {
		return err
	}
	c.AddHandler(handler)
	return nil
}

// GetChain is to get chain
func GetChain(serviceType string, name string) (*Chain, error) {
	if name == "" {
		name = common.DefaultChainName
	}
	c := &Chain{}
	origin, ok := ChainMap[serviceType+name]
	if !ok {
		return nil, fmt.Errorf("get chain [%s] failed", serviceType+name)
	}
	*c = *origin
	return c, nil
}
