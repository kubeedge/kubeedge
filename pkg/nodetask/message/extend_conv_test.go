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

package message

import (
	"reflect"
	"testing"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

func TestFormatNodeUpgradeJobExtend(t *testing.T) {
	fromVer := "v1.0.0"
	toVer := "v1.1.0"
	expected := "v1.0.0,v1.1.0"

	result := FormatNodeUpgradeJobExtend(fromVer, toVer)
	if result != expected {
		t.Errorf("FormatNodeUpgradeJobExtend() = %v, want %v", result, expected)
	}
}

func TestParseNodeUpgradeJobExtend(t *testing.T) {
	tests := []struct {
		name        string
		extend      string
		wantFromVer string
		wantToVer   string
		wantErr     bool
	}{
		{
			name:        "valid format",
			extend:      "v1.0.0,v1.1.0",
			wantFromVer: "v1.0.0",
			wantToVer:   "v1.1.0",
			wantErr:     false,
		},
		{
			name:        "invalid format",
			extend:      "v1.0.0",
			wantFromVer: "",
			wantToVer:   "",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFromVer, gotToVer, err := ParseNodeUpgradeJobExtend(tt.extend)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseNodeUpgradeJobExtend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if gotFromVer != tt.wantFromVer {
				t.Errorf("ParseNodeUpgradeJobExtend() gotFromVer = %v, want %v", gotFromVer, tt.wantFromVer)
			}
			if gotToVer != tt.wantToVer {
				t.Errorf("ParseNodeUpgradeJobExtend() gotToVer = %v, want %v", gotToVer, tt.wantToVer)
			}
		})
	}
}

func TestFormatImagePrePullJobExtend(t *testing.T) {
	statusItems := []operationsv1alpha2.ImageStatus{
		{Image: "image1:v1"},
		{Image: "image2:v2"},
	}
	expected := `[{"image":"image1:v1"},{"image":"image2:v2"}]`

	result, err := FormatImagePrePullJobExtend(statusItems)
	if err != nil {
		t.Errorf("FormatImagePrePullJobExtend() error = %v", err)
		return
	}
	if result != expected {
		t.Errorf("FormatImagePrePullJobExtend() = %v, want %v", result, expected)
	}
}

func TestParseImagePrePullJobExtend(t *testing.T) {
	tests := []struct {
		name    string
		extend  string
		want    []operationsv1alpha2.ImageStatus
		wantErr bool
	}{
		{
			name:   "valid format",
			extend: `[{"image":"image1:v1"},{"image":"image2:v2"}]`,
			want: []operationsv1alpha2.ImageStatus{
				{Image: "image1:v1"},
				{Image: "image2:v2"},
			},
			wantErr: false,
		},
		{
			name:    "invalid format",
			extend:  `invalid-json`,
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseImagePrePullJobExtend(tt.extend)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseImagePrePullJobExtend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseImagePrePullJobExtend() got = %v, want %v", got, tt.want)
			}
		})
	}
}
