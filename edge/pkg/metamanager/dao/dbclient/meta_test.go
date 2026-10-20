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

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

// TestDeleteMetaByKeyAndUID runs the conditional delete against a real sqlite database.
func TestDeleteMetaByKeyAndUID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		deleted bool
	}{
		{"same uid", `{"metadata":{"name":"cm","uid":"uid-a"}}`, true},
		{"object recreated with another uid", `{"metadata":{"name":"cm","uid":"uid-b"}}`, false},
		{"row without uid", `{"metadata":{"name":"cm"}}`, true},
		{"row that is not json", `not json`, true},
	}
	ms := NewMetaService()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ms.InsertOrUpdate(&models.Meta{Key: "default/configmap/cm", Type: "configmap", Value: test.value}); err != nil {
				t.Fatal(err)
			}
			num, err := ms.DeleteMetaByKeyAndUID("default/configmap/cm", "uid-a")
			if err != nil {
				t.Fatal(err)
			}
			if got := num == 1; got != test.deleted {
				t.Errorf("deleted = %v, want %v", got, test.deleted)
			}
			_ = ms.DeleteMetaByKey("default/configmap/cm")
		})
	}
}
