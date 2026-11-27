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

package filter

import (
	"context"
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sinformer "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	commoninformers "github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
)

var servicesGVR = v1.SchemeGroupVersion.WithResource("services")

// stubInformerManager returns a fixed InformerPair for any GVR, standing in for
// the global informers manager.
type stubInformerManager struct {
	commoninformers.Manager
	pair *commoninformers.InformerPair
	err  error
}

func (s *stubInformerManager) GetInformerPair(schema.GroupVersionResource) (*commoninformers.InformerPair, error) {
	return s.pair, s.err
}

// newServicesInformerPair builds an InformerPair for services backed by a fake
// clientset. The informer is only started, and therefore only syncs, when
// start is true.
func newServicesInformerPair(t *testing.T, start bool, objs ...*v1.Service) *commoninformers.InformerPair {
	t.Helper()

	client := kubefake.NewSimpleClientset()
	for _, obj := range objs {
		_, err := client.CoreV1().Services(obj.Namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	factory := k8sinformer.NewSharedInformerFactory(client, 0)
	genericInformer, err := factory.ForResource(servicesGVR)
	require.NoError(t, err)

	if start {
		stopCh := make(chan struct{})
		t.Cleanup(func() { close(stopCh) })
		factory.Start(stopCh)
		factory.WaitForCacheSync(stopCh)
	}

	return &commoninformers.InformerPair{
		Lister:   genericInformer.Lister(),
		Informer: genericInformer.Informer(),
	}
}

func patchInformersManager(manager commoninformers.Manager) *gomonkey.Patches {
	return gomonkey.ApplyFuncReturn(commoninformers.GetInformersManager, manager)
}

func TestGetSyncedResourceListerSynced(t *testing.T) {
	svc := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-service", Namespace: "default"}}
	pair := newServicesInformerPair(t, true, svc)

	patches := patchInformersManager(&stubInformerManager{pair: pair})
	defer patches.Reset()

	lister, err := GetSyncedResourceLister(servicesGVR)
	require.NoError(t, err)

	got, err := lister.ByNamespace("default").Get("test-service")
	require.NoError(t, err)
	assert.Equal(t, svc, got)
}

func TestGetSyncedResourceListerNotSynced(t *testing.T) {
	pair := newServicesInformerPair(t, false)

	patches := patchInformersManager(&stubInformerManager{pair: pair})
	defer patches.Reset()

	lister, err := GetSyncedResourceLister(servicesGVR)
	assert.Nil(t, lister)
	assert.ErrorContains(t, err, "has not synced yet")
}

func TestGetSyncedResourceListerManagerError(t *testing.T) {
	patches := patchInformersManager(&stubInformerManager{err: errors.New("no kind for resource")})
	defer patches.Reset()

	lister, err := GetSyncedResourceLister(servicesGVR)
	assert.Nil(t, lister)
	assert.ErrorContains(t, err, "no kind for resource")
}

// TestGetSyncedResourceListerNoInformer covers managers that report neither an
// InformerPair nor an error for an unregistered resource, which must be turned
// into an error rather than dereferenced.
func TestGetSyncedResourceListerNoInformer(t *testing.T) {
	patches := patchInformersManager(&stubInformerManager{})
	defer patches.Reset()

	lister, err := GetSyncedResourceLister(servicesGVR)
	assert.Nil(t, lister)
	assert.ErrorContains(t, err, "no informer registered")
}

func TestSameGroupAs(t *testing.T) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	nodes := map[string]string{
		"edge-a1": "site-a",
		"edge-a2": "site-a",
		"edge-b1": "site-b",
		"edge-x":  "",
		"cloud-1": "",
	}
	for name, group := range nodes {
		node := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: name}}
		if group != "" {
			node.Labels = map[string]string{nodegroup.LabelBelongingTo: group}
		}
		assert.NoError(t, indexer.Add(node))
	}
	lister := cache.NewGenericLister(indexer, v1.Resource("nodes"))
	listerCalls := 0
	patches := gomonkey.ApplyFunc(GetSyncedResourceLister, func(schema.GroupVersionResource) (cache.GenericLister, error) {
		listerCalls++
		return lister, nil
	})
	defer patches.Reset()

	cases := []struct {
		target, ep string
		want       bool
	}{
		{"edge-a1", "edge-a2", true},
		{"edge-a1", "edge-b1", false},
		{"edge-a1", "cloud-1", false},
		// A node in no group keeps only its own endpoints, not every
		// ungrouped cloud node's.
		{"edge-x", "cloud-1", false},
		{"edge-x", "edge-a1", false},
		{"edge-x", "edge-x", true},
		{"edge-a1", "missing", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, SameGroupAs(c.target)(c.ep), "%s -> %s", c.target, c.ep)
	}

	// The lister is looked up once per SameGroupAs call, not once per endpoint.
	listerCalls = 0
	inGroup := SameGroupAs("edge-a1")
	for _, ep := range []string{"edge-a2", "edge-b1", "cloud-1"} {
		inGroup(ep)
	}
	assert.Equal(t, 1, listerCalls)
}
