/*
Copyright 2022 The KubeEdge Authors.

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

package csidriver

import (
	gocontext "context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubeedge/beehive/pkg/core/model"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func Test_buildResource(t *testing.T) {
	type args struct {
		nodeID       string
		namespace    string
		resourceType string
		resourceID   string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "required parameter not set(nodeID)",
			args: args{
				nodeID:       "",
				namespace:    "default",
				resourceType: "persistentvolume",
				resourceID:   "node-id",
			},
			wantErr: true,
		},
		{
			name: "required parameter not set(namespace)",
			args: args{
				nodeID:       "node1",
				namespace:    "",
				resourceType: "persistentvolume",
				resourceID:   "node-id",
			},
			wantErr: true,
		},
		{
			name: "required parameter not set(resourceType)",
			args: args{
				nodeID:       "node1",
				namespace:    "default",
				resourceType: "",
				resourceID:   "node-id",
			},
			wantErr: true,
		},
		{
			name: "required parameter not set(resourceID)",
			args: args{
				nodeID:       "node1",
				namespace:    "default",
				resourceType: "persistentvolume",
				resourceID:   "",
			},
			want:    "node/node1/default/persistentvolume",
			wantErr: false,
		},
		{
			name: "all required parameter set",
			args: args{
				nodeID:       "node1",
				namespace:    "default",
				resourceType: "persistentvolume",
				resourceID:   "node-id",
			},
			want:    "node/node1/default/persistentvolume/node-id",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildResource(tt.args.nodeID, tt.args.namespace, tt.args.resourceType, tt.args.resourceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildResource() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_extractMessage(t *testing.T) {
	type args struct {
		context string
	}
	tests := []struct {
		name    string
		args    args
		want    *model.Message
		wantErr bool
	}{
		{
			name: "empty context",
			args: args{
				context: "",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "unmarshal context failed",
			args: args{
				context: "{\"header\": {\"msg_id\": \"hello\", \"timestamp\": \"41423\"}}",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "extract message successfully",
			args: args{
				context: "{\"header\": {\"msg_id\": \"hello\", \"timestamp\": 1465989235}, \"route\": {\"source\": \"service\"}, \"content\": \"hello world\"}",
			},
			want: &model.Message{
				Header: model.MessageHeader{
					ID:        "hello",
					Timestamp: 1465989235,
				},
				Router: model.MessageRoute{
					Source: "service",
				},
				Content: "hello world",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractMessage(tt.args.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractMessage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_logGRPC(t *testing.T) {
	type args struct {
		ctx     context.Context
		req     interface{}
		info    *grpc.UnaryServerInfo
		handler grpc.UnaryHandler
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "handler throw some error messages",
			args: args{
				ctx: context.TODO(),
				req: nil,
				info: &grpc.UnaryServerInfo{
					Server:     "127.0.0.1:9080",
					FullMethod: "Get",
				},
				handler: func(ctx gocontext.Context, req interface{}) (interface{}, error) {
					if req == nil {
						return nil, fmt.Errorf("please provide valid req")
					}
					return "hello world", nil
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "handler throw some error messages",
			args: args{
				ctx: context.TODO(),
				req: "hello",
				info: &grpc.UnaryServerInfo{
					Server:     "127.0.0.1:9080",
					FullMethod: "Get",
				},
				handler: func(ctx gocontext.Context, req interface{}) (interface{}, error) {
					if req == nil {
						return nil, fmt.Errorf("please provide valid req")
					}
					return "hello world", nil
				},
			},
			want:    "hello world",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := logGRPC(tt.args.ctx, tt.args.req, tt.args.info, tt.args.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("logGRPC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("logGRPC() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newNonBlockingGRPCServer(t *testing.T) {
	tests := []struct {
		name string
		want *nonBlockingGRPCServer
	}{
		{
			name: "new NonBlocking GRPC Server",
			want: newNonBlockingGRPCServer(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newNonBlockingGRPCServer(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newNonBlockingGRPCServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseEndpoint(t *testing.T) {
	type args struct {
		ep string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name: "has no unix or tcp prefix(empty)",
			args: args{
				ep: "",
			},
			want:    "",
			want1:   "",
			wantErr: true,
		},
		{
			name: "has no unix or tcp prefix",
			args: args{
				ep: "grpc://127.0.0.1:9080",
			},
			want:    "",
			want1:   "",
			wantErr: true,
		},
		{
			name: "has no unix or tcp prefix(invalid sep)",
			args: args{
				ep: "tcp:/127.0.0.1:9080",
			},
			want:    "",
			want1:   "",
			wantErr: true,
		},
		{
			name: "unix prefix and parsed successfully",
			args: args{
				ep: "unix://127.0.0.1:9080",
			},
			want:    "unix",
			want1:   "127.0.0.1:9080",
			wantErr: false,
		},
		{
			name: "tcp prefix and parsed successfully",
			args: args{
				ep: "tcp://127.0.0.1:9080",
			},
			want:    "tcp",
			want1:   "127.0.0.1:9080",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseEndpoint(tt.args.ep)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEndpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseEndpoint() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseEndpoint() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_nonBlockingGRPCServer_Start(t *testing.T) {
	type fields struct {
		wg     sync.WaitGroup
		server *grpc.Server
	}
	type args struct {
		endpoint string
		ids      csi.IdentityServer
		cs       csi.ControllerServer
		ns       csi.NodeServer
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "start nonBlocking GRPC Server",
			fields: fields{
				wg:     sync.WaitGroup{},
				server: grpc.NewServer(),
			},
			args: args{
				endpoint: "tcp://localhost:0",
				ids:      nil,
				cs:       nil,
				ns:       nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &nonBlockingGRPCServer{
				wg:     tt.fields.wg,
				server: tt.fields.server,
			}
			s.Start(tt.args.endpoint, tt.args.ids, tt.args.cs, tt.args.ns)
		})
	}
}

func Test_nonBlockingGRPCServer_Wait(t *testing.T) {
	type fields struct {
		wg     sync.WaitGroup
		server *grpc.Server
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "call wait function for a nonBlocking GRPC Server",
			fields: fields{
				wg:     sync.WaitGroup{},
				server: grpc.NewServer(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &nonBlockingGRPCServer{
				wg:     tt.fields.wg,
				server: tt.fields.server,
			}
			s.Wait()
		})
	}
}

func Test_nonBlockingGRPCServer_Stop(t *testing.T) {
	type fields struct {
		wg     sync.WaitGroup
		server *grpc.Server
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "stop a nonBlocking GRPC Server",
			fields: fields{
				wg:     sync.WaitGroup{},
				server: grpc.NewServer(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &nonBlockingGRPCServer{
				wg:     tt.fields.wg,
				server: tt.fields.server,
			}
			s.Stop()
		})
	}
}

func Test_nonBlockingGRPCServer_ForceStop(t *testing.T) {
	type fields struct {
		wg     sync.WaitGroup
		server *grpc.Server
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "force stop a nonBlocking GRPC Server",
			fields: fields{
				wg:     sync.WaitGroup{},
				server: grpc.NewServer(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &nonBlockingGRPCServer{
				wg:     tt.fields.wg,
				server: tt.fields.server,
			}
			s.ForceStop()
		})
	}
}

func Test_nonBlockingGRPCServer_serve(t *testing.T) {
	type fields struct {
		wg     sync.WaitGroup
		server *grpc.Server
	}
	type args struct {
		endpoint string
		ids      csi.IdentityServer
		cs       csi.ControllerServer
		ns       csi.NodeServer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "parse endpoint failed",
			fields: fields{
				wg:     sync.WaitGroup{},
				server: grpc.NewServer(),
			},
			args: args{
				endpoint: "grpc://localhost:0",
				ids:      nil,
				cs:       nil,
				ns:       nil,
			},
			wantErr: true,
		},
		{
			name: "unix requested address is not valid",
			fields: fields{
				wg:     sync.WaitGroup{},
				server: grpc.NewServer(),
			},
			args: args{
				endpoint: "unix://localhost:0",
				ids:      nil,
				cs:       nil,
				ns:       nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &nonBlockingGRPCServer{
				wg:     tt.fields.wg,
				server: tt.fields.server,
			}

			defer s.Stop()

			if err := s.serve(tt.args.endpoint, tt.args.ids, tt.args.cs, tt.args.ns); (err != nil) != tt.wantErr {
				t.Errorf("serve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sendToKubeEdge(t *testing.T) {
	type args struct {
		context          string
		kubeEdgeEndpoint string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "invalid endpoint",
			args: args{
				context:          "",
				kubeEdgeEndpoint: "",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sendToKubeEdge(tt.args.context, tt.args.kubeEdgeEndpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("sendToKubeEdge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("sendToKubeEdge() got = %v, want %v", got, tt.want)
			}
		})
	}
}
