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

package httpserver

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/global"
)

type fakeDevPanel struct {
	global.DevPanel
}

func (fakeDevPanel) GetDevice(string) (interface{}, error) {
	return &fakeDevice{}, nil
}

func (fakeDevPanel) GetModel(string) (common.DeviceModel, error) {
	return common.DeviceModel{}, errors.New("model not found")
}

type fakeDevice struct {
	Instance common.DeviceInstance
}

func TestMetaGetModelMissingModel(t *testing.T) {
	rs := &RestServer{devPanel: fakeDevPanel{}}
	w := httptest.NewRecorder()
	rs.MetaGetModel(w, httptest.NewRequest(http.MethodGet, APIMetaRoute+"/model/default/dev", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if want := "Get device model error: model not found\n"; w.Body.String() != want {
		t.Errorf("body = %q, want %q", w.Body.String(), want)
	}
}
