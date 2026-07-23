/*
Copyright 2026 The KubeEdge Authors.
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

package debug

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

// TestGetPodsFromDatabaseNoStaleKeys guards against a regression where
// json.Unmarshal reused the same map across loop iterations, leaking keys
// from one record's JSON into the next record's output.
func TestGetPodsFromDatabaseNoStaleKeys(t *testing.T) {
	dao.Init(":memory:", &v1alpha2.MetaManager{Enable: true})
	ms = dbclient.NewMetaService()

	metas := []models.Meta{
		{Key: "default/pod/pod1", Type: model.ResourceTypePod, Value: `{"metadata":{"name":"pod1"},"extra":"stale-value"}`},
		{Key: "default/podstatus/pod1", Type: model.ResourceTypePodStatus, Value: `{"Status":{"phase":"Running"}}`},
		{Key: "default/pod/pod2", Type: model.ResourceTypePod, Value: `{"metadata":{"name":"pod2"}}`},
		{Key: "default/podstatus/pod2", Type: model.ResourceTypePodStatus, Value: `{"Status":{"phase":"Pending"}}`},
	}
	for i := range metas {
		require.NoError(t, ms.SaveMeta(&metas[i]))
	}

	g := &GetOptions{AllNamespace: true}
	results, err := g.getPodsFromDatabase("default", nil)
	require.NoError(t, err)
	require.Len(t, results, 2)

	var pod2Result map[string]interface{}
	for _, r := range results {
		if r.Key == "default/pod/pod2" {
			require.NoError(t, json.Unmarshal([]byte(r.Value), &pod2Result))
		}
	}
	require.NotNil(t, pod2Result)
	_, hasStaleKey := pod2Result["extra"]
	assert.False(t, hasStaleKey, "pod2 output must not contain pod1's stale 'extra' key")
}

// TestGetNodeFromDatabaseNoStaleKeys is the node-side counterpart of
// TestGetPodsFromDatabaseNoStaleKeys.
func TestGetNodeFromDatabaseNoStaleKeys(t *testing.T) {
	dao.Init(":memory:", &v1alpha2.MetaManager{Enable: true})
	ms = dbclient.NewMetaService()

	metas := []models.Meta{
		{Key: "default/node/node1", Type: model.ResourceTypeNode, Value: `{"metadata":{"name":"node1"},"extra":"stale-value"}`},
		{Key: "default/nodestatus/node1", Type: model.ResourceTypeNodeStatus, Value: `{"Status":{"phase":"Running"}}`},
		{Key: "default/node/node2", Type: model.ResourceTypeNode, Value: `{"metadata":{"name":"node2"}}`},
		{Key: "default/nodestatus/node2", Type: model.ResourceTypeNodeStatus, Value: `{"Status":{"phase":"Pending"}}`},
	}
	for i := range metas {
		require.NoError(t, ms.SaveMeta(&metas[i]))
	}

	g := &GetOptions{AllNamespace: true}
	results, err := g.getNodeFromDatabase("default", nil)
	require.NoError(t, err)
	require.Len(t, results, 2)

	var node2Result map[string]interface{}
	for _, r := range results {
		if r.Key == "default/node/node2" {
			require.NoError(t, json.Unmarshal([]byte(r.Value), &node2Result))
		}
	}
	require.NotNil(t, node2Result)
	_, hasStaleKey := node2Result["extra"]
	assert.False(t, hasStaleKey, "node2 output must not contain node1's stale 'extra' key")
}
