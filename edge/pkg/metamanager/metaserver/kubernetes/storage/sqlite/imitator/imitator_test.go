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
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func newTestImitator() *imitator {
	return &imitator{
		versioner: Versioner,
		codec:     unstructured.UnstructuredJSONScheme,
	}
}

func newObjectMessage(operation string) model.Message {
	const configMap = `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1","namespace":"default","resourceVersion":"10"}}`
	msg := model.NewMessage("")
	msg.BuildRouter("edgecontroller", "resource", "default/configmap/cm1", operation)
	msg.Content = []byte(configMap)
	return *msg
}

func TestInjectReturnsErrorWhenWriteFails(t *testing.T) {
	s := newTestImitator()

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethodFunc(reflect.TypeOf(s), "InsertOrUpdateObj",
		func(_ context.Context, _ runtime.Object) error {
			return errors.New("database is locked")
		})

	err := s.Inject(newObjectMessage(model.InsertOperation))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database is locked")
}

func TestInjectSuccess(t *testing.T) {
	s := newTestImitator()

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethodFunc(reflect.TypeOf(s), "InsertOrUpdateObj",
		func(_ context.Context, _ runtime.Object) error {
			return nil
		})

	err := s.Inject(newObjectMessage(model.InsertOperation))
	assert.NoError(t, err)
}

func TestInsertOrUpdateObjResourceVersionError(t *testing.T) {
	s := newTestImitator()

	obj := &unstructured.Unstructured{}
	err := obj.UnmarshalJSON([]byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1","namespace":"default","resourceVersion":"not-a-number"}}`))
	require.NoError(t, err)

	err = s.InsertOrUpdateObj(context.TODO(), obj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource version")
}
