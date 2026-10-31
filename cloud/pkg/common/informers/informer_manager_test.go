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

package informers

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

var fooGVR = schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "foos"}

// newTestInformers returns an informers manager whose RESTMapper only knows
// example.com/v1 Foo, backed by a fake dynamic client.
func newTestInformers(stopCh <-chan struct{}, listReactor k8stesting.ReactionFunc) *informers {
	gvk := fooGVR.GroupVersion().WithKind("Foo")
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{gvk.GroupVersion()})
	mapper.Add(gvk, meta.RESTScopeNamespace)

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{fooGVR: "FooList"})
	if listReactor != nil {
		dynamicClient.PrependReactor("list", fooGVR.Resource, listReactor)
	}

	return &informers{
		stopCh:                 stopCh,
		mapper:                 mapper,
		customInformers:        make(map[string]cache.SharedIndexInformer),
		informersByGVR:         make(map[schema.GroupVersionResource]*InformerPair),
		dynamicInformerFactory: dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicClient, 0, v1.NamespaceAll, nil),
	}
}

func TestGetInformerPairUnknownResource(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	ifs := newTestInformers(stopCh, nil)

	gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "bars"}
	informerPair, err := ifs.GetInformerPair(gvr)

	assert.Nil(t, informerPair)
	assert.Error(t, err)
	assert.True(t, meta.IsNoMatchError(err), "expected a no match error, got %v", err)
	assert.Empty(t, ifs.informersByGVR)
}

func TestGetInformerPairSynced(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	ifs := newTestInformers(stopCh, nil)

	informerPair, err := ifs.GetInformerPair(fooGVR)

	require.NoError(t, err)
	require.NotNil(t, informerPair)
	assert.True(t, informerPair.Informer.HasSynced())
	assert.Equal(t, informerPair, ifs.informersByGVR[fooGVR])

	cached, err := ifs.GetInformerPair(fooGVR)
	assert.NoError(t, err)
	assert.Same(t, informerPair, cached)
}

func TestGetInformerPairSyncTimeout(t *testing.T) {
	origTimeout := informerSyncTimeout
	informerSyncTimeout = 2 * time.Second
	defer func() { informerSyncTimeout = origTimeout }()

	listed := make(chan struct{}, 1)
	forbidden := func(_ k8stesting.Action) (bool, runtime.Object, error) {
		select {
		case listed <- struct{}{}:
		default:
		}
		return true, nil, apierrors.NewForbidden(fooGVR.GroupResource(), "", errors.New("cloudcore cannot list foos"))
	}

	stopCh := make(chan struct{})
	defer close(stopCh)
	ifs := newTestInformers(stopCh, forbidden)

	type result struct {
		informerPair *InformerPair
		err          error
	}
	done := make(chan result, 1)
	go func() {
		informerPair, err := ifs.GetInformerPair(fooGVR)
		done <- result{informerPair, err}
	}()

	select {
	case <-listed:
	case <-time.After(informerSyncTimeout):
		t.Fatal("informer never listed foos")
	}

	// While GetInformerPair waits for the informer to sync, the lock must be free
	// so callers for other resources are not blocked.
	assert.Eventually(t, func() bool {
		if !ifs.lock.TryLock() {
			return false
		}
		ifs.lock.Unlock()
		return true
	}, time.Second, 10*time.Millisecond, "informers lock is held while waiting for the informer to sync")
	select {
	case r := <-done:
		t.Fatalf("GetInformerPair returned before the sync timeout: %v", r.err)
	default:
	}

	select {
	case r := <-done:
		assert.Nil(t, r.informerPair)
		assert.ErrorContains(t, r.err, "failed waiting for "+fooGVR.String()+" Informer to sync")
	case <-time.After(3 * informerSyncTimeout):
		t.Fatal("GetInformerPair did not give up waiting for the informer to sync")
	}

	ifs.lock.Lock()
	defer ifs.lock.Unlock()
	assert.NotContains(t, ifs.informersByGVR, fooGVR, "an informer that failed to sync must not be cached")
}
