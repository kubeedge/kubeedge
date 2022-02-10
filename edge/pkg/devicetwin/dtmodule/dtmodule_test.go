/*
Copyright 2018 The KubeEdge Authors.

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

package dtmodule_test

import (
	"reflect"
	"testing"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	. "github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmanager"
	. "github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmodule"
)

func TestDTModule_InitWorker(t *testing.T) {
	type fields struct {
		Name string
	}
	ctx, err := dtcontext.InitDTContext()
	if err != nil {
		t.Fatalf("failed to init devicetwin context: %v", err)
	}
	recvCh, confirmCh, heartBearCh := make(chan interface{}), make(chan interface{}), make(chan interface{})
	tests := []struct {
		name   string
		fields fields
		want   DTWorker
	}{
		{
			name: dtcommon.MemModule,
			fields: fields{
				Name: dtcommon.MemModule,
			},
			want: MemWorker{
				Worker: Worker{
					ReceiverChan:  recvCh,
					ConfirmChan:   confirmCh,
					HeartBeatChan: heartBearCh,
					DTContexts:    ctx,
				},
				Group: dtcommon.MemModule,
			},
		},
		{
			name: dtcommon.TwinModule,
			fields: fields{
				Name: dtcommon.TwinModule,
			},
			want: TwinWorker{
				Worker: Worker{
					ReceiverChan:  recvCh,
					ConfirmChan:   confirmCh,
					HeartBeatChan: heartBearCh,
					DTContexts:    ctx,
				},
				Group: dtcommon.TwinModule,
			},
		},
		{
			name: dtcommon.DeviceModule,
			fields: fields{
				Name: dtcommon.DeviceModule,
			},
			want: DeviceWorker{
				Worker: Worker{
					ReceiverChan:  recvCh,
					ConfirmChan:   confirmCh,
					HeartBeatChan: heartBearCh,
					DTContexts:    ctx,
				},
				Group: dtcommon.DeviceModule,
			},
		},
		{
			name: dtcommon.CommModule,
			fields: fields{
				Name: dtcommon.CommModule,
			},
			want: CommWorker{
				Worker: Worker{
					ReceiverChan:  recvCh,
					ConfirmChan:   confirmCh,
					HeartBeatChan: heartBearCh,
					DTContexts:    ctx,
				},
				Group: dtcommon.CommModule,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := &DTModule{
				Name: tt.fields.Name,
			}
			dm.InitWorker(recvCh, confirmCh, heartBearCh, ctx)
			if !reflect.DeepEqual(tt.want, dm.Worker) {
				t.Errorf("Test %v failed: expected %v, but got %v", tt.name, tt.want, dm.Worker)
			}
		})
	}
}

func TestDTModule_Start(t *testing.T) {
	type fields struct {
		Name        string
		RecvCh      chan interface{}
		ConfirmCh   chan interface{}
		HeartBeatCh chan interface{}
		Ctx         *dtcontext.DTContext
	}
	ctx, err := dtcontext.InitDTContext()
	if err != nil {
		t.Fatalf("failed to init devicetwin context: %v", err)
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "test heart beat",
			fields: fields{
				Name:        dtcommon.MemModule,
				RecvCh:      make(chan interface{}),
				ConfirmCh:   make(chan interface{}),
				HeartBeatCh: make(chan interface{}),
				Ctx:         ctx,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := DTModule{
				Name: tt.fields.Name,
			}
			dm.InitWorker(tt.fields.RecvCh, tt.fields.ConfirmCh, tt.fields.HeartBeatCh, tt.fields.Ctx)
			_, ok := ctx.ModulesHealth.Load(dtcommon.MemModule)
			if ok {
				t.Fatalf("%s not exist", dtcommon.MemModule)
			}
			go dm.Start()
			ping := "ping"
			tt.fields.HeartBeatCh <- ping
			tt.fields.HeartBeatCh <- ping
			lastTime, ok := ctx.ModulesHealth.Load(dtcommon.MemModule)
			if !ok {
				t.Fatalf("%s not exist", dtcommon.MemModule)
			}
			_, ok = lastTime.(int64)
			if !ok {
				t.Fatalf("time type is not valid: %T", lastTime)
			}
		})
	}
}
