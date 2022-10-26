/*
Copyright 2022 The KubeEdge Authors.

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

package metric

import (
	"fmt"
	"sync"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/handler"
)

const Gauge = "gauge"

var indicators sync.Map

// Register a metric function, function return name, type, help and value
func Register(indicator func(handler.Handler) (string, string, string, string), messageHandler handler.Handler) {
	name, _, _, _ := indicator(messageHandler)
	indicators.Store(name, indicator)
}

// Unregister a metric
func Unregister(indicator func(handler.Handler) (string, string, string, string), messageHandler handler.Handler) {
	name, _, _, _ := indicator(messageHandler)
	indicators.Delete(name)
}

// UnregisterAll unregister all indicators
func UnregisterAll() {
	indicators.Range(func(key, value interface{}) bool {
		indicators.Delete(key)
		return true
	})
}

// informationFormat information body for metric
func informationFormat() string {
	output := ""
	indicators.Range(func(key, value interface{}) bool {
		f, ok := value.(func() (string, string, string, string))
		if ok {
			name, T, help, v := f()
			output += fmt.Sprintf("# HELP %s %s\n# TYPE %s %s\n%s %s\n", name, help, name, T, name, v)
		}
		return true
	})
	return output
}
