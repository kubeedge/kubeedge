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

package imitator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

func newObj(apiVersion, kind, namespace, name, rv string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetNamespace(namespace)
	obj.SetName(name)
	obj.SetResourceVersion(rv)
	return obj
}

var groupSeq atomic.Int32

// setup initializes the database and returns an API group never used before,
// since the default RESTMapper keeps what it learned for the whole process.
func setup(base string) string {
	dao.Init("file:imitatortest?mode=memory&cache=shared", &v1alpha2.MetaManager{Enable: true})
	initRESTMapper(util.DefaultRESTMapper())
	return fmt.Sprintf("g%d.%s", groupSeq.Add(1), base)
}

func names(t *testing.T, key string) []string {
	resp, err := DefaultV2Client.List(context.TODO(), key)
	require.NoError(t, err)
	var ret []string
	for _, m := range *resp.Kvs {
		ret = append(ret, m.Name)
	}
	return ret
}

// TestObjectsSavedUnderGuessedResourceAreMoved reproduces an upgrade: objects
// of a kind with an irregular plural were saved under the guessed resource by a
// previous version, and must be served under the real resource once its
// discovery document, cached by the previous version, is loaded.
func TestObjectsSavedUnderGuessedResourceAreMoved(t *testing.T) {
	group := setup("istio.io")
	gv := group + "/v1alpha3"
	ctx := context.TODO()

	// Saved before the mapping is known: Gateway is guessed as gatewaies.
	require.NoError(t, DefaultV2Client.InsertOrUpdateObj(ctx, newObj(gv, "Gateway", "default", "gw1", "10")))
	require.NoError(t, DefaultV2Client.InsertOrUpdateObj(ctx, newObj(gv, "Gateway", "default", "gw2", "11")))
	assert.ElementsMatch(t, []string{"gw1", "gw2"}, names(t, "/"+gv+"/gatewaies/null/null"))
	assert.Empty(t, names(t, "/"+gv+"/gateways/null/null"))

	doc, err := json.Marshal(&metav1.APIResourceList{
		TypeMeta:     metav1.TypeMeta{Kind: "APIResourceList", APIVersion: "v1"},
		GroupVersion: gv,
		APIResources: []metav1.APIResource{{Name: "gateways", Namespaced: true, Kind: "Gateway"}},
	})
	require.NoError(t, err)
	require.NoError(t, DefaultV2Client.InsertOrUpdatePassThroughObj(ctx, doc, "/apis/"+gv))

	// What StorageInit does when edgecore starts.
	initRESTMapper(util.DefaultRESTMapper())

	assert.Empty(t, names(t, "/"+gv+"/gatewaies/null/null"))
	assert.ElementsMatch(t, []string{"gw1", "gw2"}, names(t, "/"+gv+"/gateways/null/null"))
	obj, err := DefaultV2Client.Get(ctx, "/"+gv+"/gateways/default/gw1")
	require.NoError(t, err)
	assert.Len(t, *obj.Kvs, 1)

	// Objects saved from now on use the real resource as well.
	require.NoError(t, DefaultV2Client.InsertOrUpdateObj(ctx, newObj(gv, "Gateway", "default", "gw3", "12")))
	assert.ElementsMatch(t, []string{"gw1", "gw2", "gw3"}, names(t, "/"+gv+"/gateways/null/null"))

	// Running StorageInit again, e.g. after a module restart, is harmless.
	initRESTMapper(util.DefaultRESTMapper())
	assert.ElementsMatch(t, []string{"gw1", "gw2", "gw3"}, names(t, "/"+gv+"/gateways/null/null"))
}

// TestMigrateResourceOnlyMovesObjectsOfTheKind checks that objects of another
// kind saved under the old resource stay where they are.
func TestMigrateResourceOnlyMovesObjectsOfTheKind(t *testing.T) {
	group := setup("example.com")
	gv := group + "/v1"
	ctx := context.TODO()

	// Mailbox is guessed as mailboxs.
	require.NoError(t, DefaultV2Client.InsertOrUpdateObj(ctx, newObj(gv, "Mailbox", "default", "box", "20")))
	stranger, err := json.Marshal(newObj(gv, "Stranger", "default", "stranger", "21"))
	require.NoError(t, err)
	oldGVR := schema.GroupVersionResource{Group: group, Version: "v1", Resource: "mailboxs"}
	require.NoError(t, dbclient.NewMetaV2Service().InsertOrReplaceMetaV2(&models.MetaV2{
		Key:                  "/" + gv + "/mailboxs/default/stranger",
		GroupVersionResource: oldGVR.String(),
		Namespace:            "default",
		Name:                 "stranger",
		Value:                string(stranger),
	}))
	assert.ElementsMatch(t, []string{"box", "stranger"}, names(t, "/"+gv+"/mailboxs/null/null"))

	util.DefaultRESTMapper().RegisterMapping(oldGVR.GroupVersion().WithResource("mailboxes"), "Mailbox")

	assert.ElementsMatch(t, []string{"stranger"}, names(t, "/"+gv+"/mailboxs/null/null"))
	assert.ElementsMatch(t, []string{"box"}, names(t, "/"+gv+"/mailboxes/null/null"))
}

// TestMigrateResourceKeepsNewerObject checks the conflict case of the
// migration: an old object saved under the guessed resource, and a newer one
// of the same name already saved under the real resource. The newer object is
// kept and the old row is removed.
func TestMigrateResourceKeepsNewerObject(t *testing.T) {
	group := setup("conflict.istio.io")
	gv := group + "/v1alpha3"
	ctx := context.TODO()
	gatewaies := schema.GroupVersionResource{Group: group, Version: "v1alpha3", Resource: "gatewaies"}
	gateways := gatewaies.GroupVersion().WithResource("gateways")

	// Saved by a previous version under the guessed resource.
	old := newObj(gv, "Gateway", "default", "gw", "10")
	old.SetLabels(map[string]string{"version": "old"})
	require.NoError(t, DefaultV2Client.InsertOrUpdateObj(ctx, old))
	// Saved again under the real resource, e.g. read back from the cloud.
	newer := newObj(gv, "Gateway", "default", "gw", "20")
	newer.SetLabels(map[string]string{"version": "new"})
	value, err := json.Marshal(newer)
	require.NoError(t, err)
	require.NoError(t, dbclient.NewMetaV2Service().InsertOrReplaceMetaV2(&models.MetaV2{
		Key:                  "/" + gv + "/gateways/default/gw",
		GroupVersionResource: gateways.String(),
		Namespace:            "default",
		Name:                 "gw",
		ResourceVersion:      20,
		Value:                string(value),
	}))
	require.Equal(t, []string{"gw"}, names(t, "/"+gv+"/gatewaies/null/null"))
	require.Equal(t, []string{"gw"}, names(t, "/"+gv+"/gateways/null/null"))

	util.DefaultRESTMapper().RegisterMapping(gateways, "Gateway")

	// The old row is removed.
	assert.Empty(t, names(t, "/"+gv+"/gatewaies/null/null"))
	// The newer object is kept.
	resp, err := DefaultV2Client.Get(ctx, "/"+gv+"/gateways/default/gw")
	require.NoError(t, err)
	require.Len(t, *resp.Kvs, 1)
	kept := (*resp.Kvs)[0]
	assert.Equal(t, uint64(20), kept.ResourceVersion)
	var obj unstructured.Unstructured
	require.NoError(t, json.Unmarshal([]byte(kept.Value), &obj.Object))
	assert.Equal(t, "new", obj.GetLabels()["version"])
	assert.Equal(t, "20", obj.GetResourceVersion())
}

// TestMigrateResourceReplacesOlderObject checks the other conflict case: the
// copy saved under the real resource is older than the one being moved, so the
// moved object replaces it.
func TestMigrateResourceReplacesOlderObject(t *testing.T) {
	group := setup("older.istio.io")
	gv := group + "/v1alpha3"
	ctx := context.TODO()
	gateways := schema.GroupVersionResource{Group: group, Version: "v1alpha3", Resource: "gateways"}

	moved := newObj(gv, "Gateway", "default", "gw", "30")
	moved.SetLabels(map[string]string{"version": "new"})
	require.NoError(t, DefaultV2Client.InsertOrUpdateObj(ctx, moved))
	older := newObj(gv, "Gateway", "default", "gw", "5")
	older.SetLabels(map[string]string{"version": "old"})
	value, err := json.Marshal(older)
	require.NoError(t, err)
	require.NoError(t, dbclient.NewMetaV2Service().InsertOrReplaceMetaV2(&models.MetaV2{
		Key:                  "/" + gv + "/gateways/default/gw",
		GroupVersionResource: gateways.String(),
		Namespace:            "default",
		Name:                 "gw",
		ResourceVersion:      5,
		Value:                string(value),
	}))

	util.DefaultRESTMapper().RegisterMapping(gateways, "Gateway")

	assert.Empty(t, names(t, "/"+gv+"/gatewaies/null/null"))
	resp, err := DefaultV2Client.Get(ctx, "/"+gv+"/gateways/default/gw")
	require.NoError(t, err)
	require.Len(t, *resp.Kvs, 1)
	assert.Equal(t, uint64(30), (*resp.Kvs)[0].ResourceVersion)
	var obj unstructured.Unstructured
	require.NoError(t, json.Unmarshal([]byte((*resp.Kvs)[0].Value), &obj.Object))
	assert.Equal(t, "new", obj.GetLabels()["version"])
}

// duringMigration runs op in a goroutine while the next migration is in
// progress, i.e. after the mapping change and before the objects are moved,
// and returns a channel closed when op has returned.
func duringMigration(t *testing.T, op func()) <-chan struct{} {
	t.Helper()
	done := make(chan struct{})
	var once sync.Once
	beforeMigrate = func() {
		once.Do(func() {
			go func() {
				defer close(done)
				op()
			}()
			// Give op the time to run if nothing prevents it.
			select {
			case <-done:
			case <-time.After(200 * time.Millisecond):
			}
		})
	}
	t.Cleanup(func() { beforeMigrate = func() {} })
	return done
}

// TestDeleteDuringMigrationDoesNotResurrect reproduces a delete event processed
// between the mapping change and the migration: the delete must not miss the
// object, which the migration would otherwise move to the new resource.
func TestDeleteDuringMigrationDoesNotResurrect(t *testing.T) {
	group := setup("delete.istio.io")
	gv := group + "/v1alpha3"
	ctx := context.TODO()

	// 1. The object exists under the guessed resource.
	gw := newObj(gv, "Gateway", "default", "gw", "10")
	require.NoError(t, DefaultV2Client.InsertOrUpdateObj(ctx, gw))
	require.Equal(t, []string{"gw"}, names(t, "/"+gv+"/gatewaies/null/null"))

	// 3. A delete event for it is processed while the migration is in progress.
	var deleteErr error
	deleted := duringMigration(t, func() {
		deleteErr = DefaultV2Client.DeleteObj(ctx, newObj(gv, "Gateway", "default", "gw", "11"))
	})

	// 2. The mapping changes Gateway -> gateways.
	util.DefaultRESTMapper().RegisterMapping(schema.GroupVersionResource{Group: group, Version: "v1alpha3", Resource: "gateways"}, "Gateway")
	<-deleted
	require.NoError(t, deleteErr)

	// The object is gone under both resources.
	assert.Empty(t, names(t, "/"+gv+"/gatewaies/null/null"))
	assert.Empty(t, names(t, "/"+gv+"/gateways/null/null"), "the deleted object was resurrected by the migration")
}

// TestInsertDuringMigrationUsesNewResource checks that an object written while
// a migration is in progress ends up under the new resource, and is not left
// behind under the old one.
func TestInsertDuringMigrationUsesNewResource(t *testing.T) {
	group := setup("insert.istio.io")
	gv := group + "/v1alpha3"
	ctx := context.TODO()

	require.NoError(t, DefaultV2Client.InsertOrUpdateObj(ctx, newObj(gv, "Gateway", "default", "gw", "10")))

	var insertErr error
	inserted := duringMigration(t, func() {
		insertErr = DefaultV2Client.InsertOrUpdateObj(ctx, newObj(gv, "Gateway", "default", "gw", "11"))
	})
	util.DefaultRESTMapper().RegisterMapping(schema.GroupVersionResource{Group: group, Version: "v1alpha3", Resource: "gateways"}, "Gateway")
	<-inserted
	require.NoError(t, insertErr)

	assert.Empty(t, names(t, "/"+gv+"/gatewaies/null/null"))
	resp, err := DefaultV2Client.Get(ctx, "/"+gv+"/gateways/default/gw")
	require.NoError(t, err)
	require.Len(t, *resp.Kvs, 1)
	assert.Equal(t, uint64(11), (*resp.Kvs)[0].ResourceVersion)
}

// TestListKindsSavedAndLoaded checks that a list kind which differs from
// <kind>List is saved, and restored after a restart.
func TestListKindsSavedAndLoaded(t *testing.T) {
	group := setup("listkind.example.com")
	databases := schema.GroupVersionResource{Group: group, Version: "v1", Resource: "databases"}
	widgets := schema.GroupVersionResource{Group: group, Version: "v1", Resource: "widgets"}

	mapper := util.DefaultRESTMapper()
	mapper.RegisterMapping(databases, "Database")
	mapper.RegisterListKind(databases, "DatabaseCollection")
	mapper.RegisterMapping(widgets, "Widget")
	mapper.RegisterListKind(widgets, "WidgetList")

	saved, err := dbclient.NewMetaV2Service().ListMetaV2ByKeyPrefix(listKindKeyPrefix + group + "/")
	require.NoError(t, err)
	require.Len(t, *saved, 1, "only list kinds which differ from <kind>List are saved")
	assert.Equal(t, "DatabaseCollection", (*saved)[0].Value)

	// After a restart, before any list is returned by the cloud: the kind is
	// known from the cached discovery documents, the list kind is not.
	restarted := util.NewRESTMapper()
	restarted.RegisterMapping(databases, "Database")
	assert.Equal(t, "DatabaseList", restarted.ListKindFor(databases).Kind)
	assert.NotZero(t, loadListKinds(restarted))
	assert.Equal(t, "DatabaseCollection", restarted.ListKindFor(databases).Kind)

	// Saved list kinds are not mistaken for discovery documents.
	docs, err := dbclient.NewMetaV2Service().ListDiscoveryMetaV2()
	require.NoError(t, err)
	for _, doc := range *docs {
		assert.NotContains(t, doc.Key, listKindKeyPrefix)
	}
}
