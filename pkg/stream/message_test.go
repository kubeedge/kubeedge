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

package stream

import (
	"reflect"
	"testing"
)

func TestMessage(t *testing.T) {
	type args struct {
		id       uint64
		messType MessageType
		data     []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			args: args{
				id:       1,
				messType: 3,
				data:     []byte("message"),
			},
			want: []byte("\x01\x03message"),
		},
		{
			args: args{
				id:       2,
				messType: 4,
				data:     []byte("message message message message"),
			},
			want: []byte("\x02\x04message message message message"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMessage(tt.args.id, tt.args.messType, tt.args.data).Bytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMessage().Bytes() = %q, want %q", got, tt.want)
			}
		})
	}
}
