/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
