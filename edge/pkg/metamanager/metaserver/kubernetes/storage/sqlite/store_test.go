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

package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apiserver/pkg/storage"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/fake"
)

const testListKey = "/core/v1/configmaps/default/null"

func newTestStore(listFn func(ctx context.Context, key string) (imitator.Resp, error)) *store {
	return &store{
		client:    fake.Client{ListF: listFn},
		versioner: imitator.Versioner,
		codec:     unstructured.UnstructuredJSONScheme,
	}
}

// TestGetListEmptyResult covers the case where the local store holds no objects
// for the requested key. This must be reported as an empty list rather than as
// an error, and the list still has to carry a resource version so that a client
// can start a watch from it.
func TestGetListEmptyResult(t *testing.T) {
	empty := make([]models.MetaV2, 0)
	s := newTestStore(func(_ context.Context, _ string) (imitator.Resp, error) {
		return imitator.Resp{Kvs: &empty, Revision: 42}, nil
	})

	list := &unstructured.UnstructuredList{}
	err := s.GetList(context.TODO(), testListKey, storage.ListOptions{Predicate: storage.Everything}, list)

	assert.NoError(t, err)
	assert.Empty(t, list.Items)
	assert.Equal(t, "42", list.GetResourceVersion())
	assert.NotEmpty(t, list.GetKind())
}

// TestGetListReturnsClientError checks that a failure from the underlying client
// is propagated instead of being reported as an empty list.
func TestGetListReturnsClientError(t *testing.T) {
	expected := errors.New("db unavailable")
	s := newTestStore(func(_ context.Context, _ string) (imitator.Resp, error) {
		return imitator.Resp{}, expected
	})

	list := &unstructured.UnstructuredList{}
	err := s.GetList(context.TODO(), testListKey, storage.ListOptions{Predicate: storage.Everything}, list)

	assert.ErrorIs(t, err, expected)
}

// TestGetListDecodesStoredObjects checks the non-empty path still works.
func TestGetListDecodesStoredObjects(t *testing.T) {
	stored := []models.MetaV2{{
		Value: `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1","namespace":"default"}}`,
	}}
	s := newTestStore(func(_ context.Context, _ string) (imitator.Resp, error) {
		return imitator.Resp{Kvs: &stored, Revision: 7}, nil
	})

	list := &unstructured.UnstructuredList{}
	err := s.GetList(context.TODO(), testListKey, storage.ListOptions{Predicate: storage.Everything}, list)

	assert.NoError(t, err)
	assert.Len(t, list.Items, 1)
	assert.Equal(t, "cm1", list.Items[0].GetName())
	assert.Equal(t, "7", list.GetResourceVersion())
}
