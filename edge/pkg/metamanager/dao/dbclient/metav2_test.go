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

package dbclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

func newTestMetaV2Service(t *testing.T) *MetaV2Service {
	dao.Init("file:metav2test?mode=memory&cache=shared", &v1alpha2.MetaManager{Enable: true})
	s := NewMetaV2Service()
	require.NoError(t, s.db.Where("1 = 1").Delete(&models.MetaV2{}).Error)
	return s
}

func TestListDiscoveryMetaV2(t *testing.T) {
	s := newTestMetaV2Service(t)
	for _, key := range []string{"/api", "/api/v1", "/apis", "/apis/cilium.io/v2", "/version", "/openapi/v2", "/openapi/v3/apis/apps/v1"} {
		require.NoError(t, s.InsertOrReplaceMetaV2(&models.MetaV2{Key: key, Value: "doc"}))
	}
	require.NoError(t, s.InsertOrReplaceMetaV2(&models.MetaV2{
		Key:                  "/core/v1/pods/default/foo",
		GroupVersionResource: schema.GroupVersionResource{Version: "v1", Resource: "pods"}.String(),
		Namespace:            "default",
		Name:                 "foo",
		Value:                "pod",
	}))

	docs, err := s.ListDiscoveryMetaV2()
	require.NoError(t, err)
	var keys []string
	for _, doc := range *docs {
		keys = append(keys, doc.Key)
	}
	assert.ElementsMatch(t, []string{"/api", "/api/v1", "/apis", "/apis/cilium.io/v2"}, keys)
}

func TestMoveMetaV2(t *testing.T) {
	s := newTestMetaV2Service(t)
	oldGVR := schema.GroupVersionResource{Group: "networking.istio.io", Version: "v1alpha3", Resource: "gatewaies"}
	newGVR := oldGVR.GroupVersion().WithResource("gateways")
	row := func(gvr schema.GroupVersionResource, name, value string, rv uint64) models.MetaV2 {
		return models.MetaV2{
			Key:                  "/" + gvr.Group + "/" + gvr.Version + "/" + gvr.Resource + "/default/" + name,
			GroupVersionResource: gvr.String(),
			Namespace:            "default",
			Name:                 name,
			ResourceVersion:      rv,
			Value:                value,
		}
	}
	for _, m := range []models.MetaV2{
		// a is only saved under the old resource.
		row(oldGVR, "a", "old a", 10),
		// b is saved under both, the copy under the new resource is newer.
		row(oldGVR, "b", "old b", 10),
		row(newGVR, "b", "new b", 20),
		// c is saved under both, the copy under the old resource is newer.
		row(oldGVR, "c", "new c", 30),
		row(newGVR, "c", "old c", 5),
	} {
		require.NoError(t, s.InsertOrReplaceMetaV2(&m))
	}

	require.NoError(t, s.MoveMetaV2(
		[]string{row(oldGVR, "a", "", 0).Key, row(oldGVR, "b", "", 0).Key, row(oldGVR, "c", "", 0).Key},
		[]models.MetaV2{row(newGVR, "a", "old a", 10), row(newGVR, "b", "old b", 10), row(newGVR, "c", "new c", 30)},
	))

	old, err := s.RawMetaByGVRNN(oldGVR, models.NullNamespace, models.NullName)
	require.NoError(t, err)
	assert.Empty(t, *old)

	moved, err := s.RawMetaByGVRNN(newGVR, models.NullNamespace, models.NullName)
	require.NoError(t, err)
	values := map[string]string{}
	for _, m := range *moved {
		values[m.Name] = m.Value
	}
	assert.Equal(t, map[string]string{"a": "old a", "b": "new b", "c": "new c"}, values)

	// Nothing to move.
	assert.NoError(t, s.MoveMetaV2(nil, nil))
}
