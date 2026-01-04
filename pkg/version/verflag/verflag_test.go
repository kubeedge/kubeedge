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

package verflag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionValue_IsBoolFlag(t *testing.T) {
	v := VersionFalse
	assert.True(t, v.IsBoolFlag())
}

func TestVersionValue_Get(t *testing.T) {
	v := VersionTrue
	assert.Equal(t, VersionTrue, v.Get())
}

func TestVersionValue_Set(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    VersionValue
		wantErr bool
	}{
		{
			name:    "Set raw",
			input:   "raw",
			want:    VersionRaw,
			wantErr: false,
		},
		{
			name:    "Set true",
			input:   "true",
			want:    VersionTrue,
			wantErr: false,
		},
		{
			name:    "Set false",
			input:   "false",
			want:    VersionFalse,
			wantErr: false,
		},
		{
			name:    "Set 1",
			input:   "1",
			want:    VersionTrue,
			wantErr: false,
		},
		{
			name:    "Set 0",
			input:   "0",
			want:    VersionFalse,
			wantErr: false,
		},
		{
			name:    "Set invalid",
			input:   "invalid",
			want:    VersionFalse,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v VersionValue
			err := v.Set(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, v)
		})
	}
}

func TestVersionValue_String(t *testing.T) {
	tests := []struct {
		name string
		v    VersionValue
		want string
	}{
		{
			name: "String raw",
			v:    VersionRaw,
			want: "raw",
		},
		{
			name: "String true",
			v:    VersionTrue,
			want: "true",
		},
		{
			name: "String false",
			v:    VersionFalse,
			want: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.v.String())
		})
	}
}

func TestVersionValue_Type(t *testing.T) {
	v := VersionFalse
	assert.Equal(t, "version", v.Type())
}
