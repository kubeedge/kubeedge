/*
Copyright 2025 The KubeEdge Authors.

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

package handlerfactory

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage"
)

func TestRestart(t *testing.T) {
	patch := gomonkey.NewPatches()
	defer patch.Reset()

	f := NewFactory()

	mockRestartResponse := &types.RestartResponse{}

	patch.ApplyFunc(limitedReadBody, func(req *http.Request, size int64) ([]byte, error) {
		if req.Body == nil {
			return []byte("[]"), nil
		}
		data, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return data, nil
	})

	patch.ApplyMethod(reflect.TypeOf(f.storage), "Restart",
		func(_ *storage.REST, _ context.Context, _ common.RestartInfo) *types.RestartResponse {
			return mockRestartResponse
		})

	originalMarshal := json.Marshal

	patch.ApplyFunc(json.Marshal, func(v interface{}) ([]byte, error) {
		if _, ok := v.(*types.RestartResponse); ok {
			return []byte(`{"success":true,"message":"Pods restarted successfully"}`), nil
		}
		return originalMarshal(v)
	})

	t.Run("Successful restart", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/restart", strings.NewReader(`["pod1", "pod2"]`))
		w := httptest.NewRecorder()

		handler := f.Restart("default")
		handler.ServeHTTP(w, req)

		resp := w.Result()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		assert.JSONEq(t, `{"success":true,"message":"Pods restarted successfully"}`, string(body))
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		invalidJSONPatch := gomonkey.ApplyFunc(limitedReadBody, func(_ *http.Request, _ int64) ([]byte, error) {
			return []byte("invalid json"), nil
		})
		defer invalidJSONPatch.Reset()

		jsonUnmarshalPatch := gomonkey.ApplyFunc(json.Unmarshal, func(data []byte, v interface{}) error {
			if string(data) == "invalid json" {
				return errors.New("invalid character 'i' looking for beginning of value")
			}
			return nil
		})
		defer jsonUnmarshalPatch.Reset()

		req := httptest.NewRequest("POST", "/restart", strings.NewReader("invalid json"))
		w := httptest.NewRecorder()

		handler := f.Restart("default")
		handler.ServeHTTP(w, req)

		resp := w.Result()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Contains(t, string(body), "invalid character")
	})

	t.Run("Marshal error", func(t *testing.T) {
		marshalErrorPatch := gomonkey.ApplyFunc(json.Marshal, func(v interface{}) ([]byte, error) {
			if _, ok := v.(*types.RestartResponse); ok {
				return nil, errors.New("marshal error")
			}
			return nil, nil
		})
		defer marshalErrorPatch.Reset()

		req := httptest.NewRequest("POST", "/restart", strings.NewReader(`["pod1"]`))
		w := httptest.NewRecorder()

		handler := f.Restart("default")
		handler.ServeHTTP(w, req)

		resp := w.Result()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Contains(t, string(body), "marshal error")
	})
}
