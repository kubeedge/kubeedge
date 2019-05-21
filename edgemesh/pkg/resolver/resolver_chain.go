package resolver

import (
	"container/list"

	"github.com/go-chassis/go-chassis/core/invocation"
)

var ResolverChain *list.List

func init() {
	ResolverChain = list.New()
}

// Resolve will loop the resolverchain to resolve the request
func Resolve(request chan []byte, stop chan interface{}, invCallback func(string, invocation.Invocation)) (invocation.Invocation, bool) {
	for resolver := ResolverChain.Front(); resolver != nil; resolver = resolver.Next() {
		inv, isFired := resolver.Value.(Resolver).Resolve(request, stop, invCallback)
		if isFired {
			return inv, true
		}
	}
	return invocation.Invocation{}, false
}

func RegisterResolver(resolver Resolver) {
	ResolverChain.PushBack(resolver)
}
