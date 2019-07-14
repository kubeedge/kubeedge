package resolver

import (
	"github.com/go-chassis/go-chassis/core/invocation"
)

type Resolver interface {
	Resolve(chan []byte, chan interface{}, func(string, invocation.Invocation, []string, bool)) (invocation.Invocation, bool)
}