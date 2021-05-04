/*
Copyright 2020 The KubeEdge Authors.
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

package debug

import (
	"reflect"
	"testing"
)

func TestSplitSelectorParameters(t *testing.T) {
	type args struct {
		args string
	}
	tests := []struct {
		name    string
		args    args
		want    []Selector
		wantErr bool
	}{
		{
			name: "testWithAllLabels",
			args: args{args: "key1==value1,key2!=value2,key3=value3"},
			want: []Selector{
				{Key: "key1", Value: "value1", Exist: true},
				{Key: "key2", Value: "value2", Exist: false},
				{Key: "key3", Value: "value3", Exist: true},
			},
			wantErr: false,
		}, {
			name:    "testWithoutLabel",
			args:    args{args: "key1"},
			want:    []Selector{},
			wantErr: false,
		}, {
			name:    "testWithEmptyValue",
			args:    args{args: "key1!="},
			want:    []Selector{{Key: "key1", Value: "", Exist: false}},
			wantErr: false,
		}, {
			name:    "testWithMoreThanOneLabel",
			args:    args{args: "key1=value1=,key2=value2"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitSelectorParameters(tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitSelectorParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitSelectorParameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
