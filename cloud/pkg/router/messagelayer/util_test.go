/*
Copyright 2021 The KubeEdge Authors.

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

package messagelayer

import (
	"fmt"
	"reflect"
	"testing"
)

func TestBuildResourceForRouter(t *testing.T) {
	type args struct {
		namespace    string
		resourceType string
		resourceID   string
	}
	tests := []struct {
		name         string
		args         args
		wantResource string
		wantErr      error
	}{
		{
			name: "case 1: empty resourceType",
			args: args{
				namespace:    "default",
				resourceType: "",
				resourceID:   "id",
			},
			wantResource: "",
			wantErr:      fmt.Errorf("required parameter are not set (resourceID or resource type)"),
		},
		{
			name: "case 2: empty resourceID",
			args: args{
				namespace:    "default",
				resourceType: "type",
				resourceID:   "",
			},
			wantResource: "",
			wantErr:      fmt.Errorf("required parameter are not set (resourceID or resource type)"),
		},
		{
			name: "case 3: empty namespace",
			args: args{
				namespace:    "",
				resourceType: "type",
				resourceID:   "id",
			},
			wantResource: "node/nodeid/default/type/id",
			wantErr:      nil,
		},
		{
			name: "case 4: success",
			args: args{
				namespace:    "default",
				resourceType: "type",
				resourceID:   "id",
			},
			wantResource: "node/nodeid/default/type/id",
			wantErr:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResource, err := BuildResourceForRouter(tt.args.namespace, tt.args.resourceType, tt.args.resourceID)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("%v: BuildResourceForRouter() error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if gotResource != tt.wantResource {
				t.Errorf("%v: BuildResourceForRouter() = %v, want %v", tt.name, gotResource, tt.wantResource)
			}
		})
	}
}
