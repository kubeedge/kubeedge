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

	"github.com/stretchr/testify/require"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

// initTestMetaDB brings up an in-memory sqlite DB for the metamanager
// dao package and points the package-level ms client at it, so
// getPodsFromDatabase/getNodeFromDatabase can be exercised against real
// query results.
func initTestMetaDB(t *testing.T) {
	dao.Init("file::memory:?cache=shared", &v1alpha2.MetaManager{Enable: true})
	ms = dbclient.NewMetaService()
	db := dao.GetDB()
	require.NoError(t, db.Where("1 = 1").Delete(&models.Meta{}).Error)
}

func TestGetPodsFromDatabase_NoStaleFieldsAcrossRecords(t *testing.T) {
	initTestMetaDB(t)
	g := &GetOptions{Namespace: "default"}

	require.NoError(t, ms.SaveMeta(&models.Meta{
		Key:   "default/pod/pod1",
		Type:  model.ResourceTypePod,
		Value: `{"metadata":{"name":"pod1"},"extraField":"should-not-leak"}`,
	}))
	require.NoError(t, ms.SaveMeta(&models.Meta{
		Key:   "default/podstatus/pod1",
		Type:  model.ResourceTypePodStatus,
		Value: `{"Status":{"phase":"Running"}}`,
	}))

	require.NoError(t, ms.SaveMeta(&models.Meta{
		Key:   "default/pod/pod2",
		Type:  model.ResourceTypePod,
		Value: `{"metadata":{"name":"pod2"}}`,
	}))
	require.NoError(t, ms.SaveMeta(&models.Meta{
		Key:   "default/podstatus/pod2",
		Type:  model.ResourceTypePodStatus,
		Value: `{"Status":{"phase":"Pending"}}`,
	}))

	results, err := g.getPodsFromDatabase("default", nil)
	require.NoError(t, err)
	require.Len(t, results, 2)

	byKey := make(map[string]models.Meta)
	for _, r := range results {
		byKey[r.Key] = r
	}

	var pod2 map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(byKey["default/pod/pod2"].Value), &pod2))
	_, leaked := pod2["extraField"]
	require.False(t, leaked, "pod2 output must not contain pod1's extraField")
}

func TestGetNodeFromDatabase_NoStaleFieldsAcrossRecords(t *testing.T) {
	initTestMetaDB(t)
	g := &GetOptions{Namespace: "default"}

	require.NoError(t, ms.SaveMeta(&models.Meta{
		Key:   "default/node/node1",
		Type:  model.ResourceTypeNode,
		Value: `{"metadata":{"name":"node1"},"extraField":"should-not-leak"}`,
	}))
	require.NoError(t, ms.SaveMeta(&models.Meta{
		Key:   "default/nodestatus/node1",
		Type:  model.ResourceTypeNodeStatus,
		Value: `{"Status":{"phase":"Running"}}`,
	}))

	require.NoError(t, ms.SaveMeta(&models.Meta{
		Key:   "default/node/node2",
		Type:  model.ResourceTypeNode,
		Value: `{"metadata":{"name":"node2"}}`,
	}))
	require.NoError(t, ms.SaveMeta(&models.Meta{
		Key:   "default/nodestatus/node2",
		Type:  model.ResourceTypeNodeStatus,
		Value: `{"Status":{"phase":"Pending"}}`,
	}))

	results, err := g.getNodeFromDatabase("default", nil)
	require.NoError(t, err)
	require.Len(t, results, 2)

	byKey := make(map[string]models.Meta)
	for _, r := range results {
		byKey[r.Key] = r
	}

	var node2 map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(byKey["default/node/node2"].Value), &node2))
	_, leaked := node2["extraField"]
	require.False(t, leaked, "node2 output must not contain node1's extraField")
}
