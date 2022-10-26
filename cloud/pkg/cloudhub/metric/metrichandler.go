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

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/handler"
)

// PreNodeCount save pre node_count
var PreNodeCount int32

// NodeCountMetric is the function register to metric return count of nodes connected
func NodeCountMetric(messageHandler handler.Handler) (string, string, string, string) {
	nodeCount := messageHandler.GetNodeNumber()
	return "node_count", Gauge, "number of nodes connected", fmt.Sprintf("%d", nodeCount)
}

// NodeCountDiffRateMetric is the function register to metric return different count rate of nodes
func NodeCountDiffRateMetric(messageHandler handler.Handler) (string, string, string, string) {
	nodeCount := messageHandler.GetNodeNumber()
	nodeCountDiffRate := 0
	if PreNodeCount != 0 {
		nodeCountDiffRate = int((float64(PreNodeCount-nodeCount) / float64(PreNodeCount)) * 100.0)
	}
	PreNodeCount = nodeCount
	return "node_count_diff_rate", Gauge, "node count difference rate", fmt.Sprintf("%d", nodeCountDiffRate)
}

func RegisterMetricHandler(messageHandler handler.Handler) {
	Register(NodeCountMetric, messageHandler)
	Register(NodeCountDiffRateMetric, messageHandler)
}
