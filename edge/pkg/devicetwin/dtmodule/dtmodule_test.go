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

package dtmodule

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmanager"
)

// mockPanicWorker implements a worker that panics for testing
type mockPanicWorker struct {
	dtmanager.Worker
}

func (w mockPanicWorker) Start() {
	panic("simulated panic for testing recovery")
}

func TestDTModule_InitWorker_DMIModule(t *testing.T) {
	ctx, err := dtcontext.InitDTContext()
	if err != nil {
		t.Fatalf("failed to init devicetwin context: %v", err)
	}
	recvCh, confirmCh, heartBeatCh := make(chan interface{}), make(chan interface{}), make(chan interface{})

	dm := &DTModule{
		Name: dtcommon.DMIModule,
	}
	dm.InitWorker(recvCh, confirmCh, heartBeatCh, ctx)

	dmiWorker, ok := dm.Worker.(dtmanager.DMIWorker)
	if !ok {
		t.Errorf("Expected worker type DMIWorker, got %T", dm.Worker)
	}

	if dmiWorker.Group != dtcommon.DMIModule {
		t.Errorf("Expected group %s, got %s", dtcommon.DMIModule, dmiWorker.Group)
	}

	if dmiWorker.ReceiverChan != recvCh {
		t.Error("ReceiverChan not properly initialized")
	}
	if dmiWorker.ConfirmChan != confirmCh {
		t.Error("ConfirmChan not properly initialized")
	}
	if dmiWorker.HeartBeatChan != heartBeatCh {
		t.Error("HeartBeatChan not properly initialized")
	}
	if dmiWorker.DTContexts != ctx {
		t.Error("DTContexts not properly initialized")
	}
}

func TestDTModule_Start_PanicRecovery(t *testing.T) {
	ctx, err := dtcontext.InitDTContext()
	if err != nil {
		t.Fatalf("failed to init devicetwin context: %v", err)
	}

	// Create channels
	recvCh, confirmCh, heartBeatCh := make(chan interface{}), make(chan interface{}), make(chan interface{})

	// Create DTModule with mock worker
	dm := &DTModule{
		Name: "TestModule",
		Worker: mockPanicWorker{
			Worker: dtmanager.Worker{
				ReceiverChan:  recvCh,
				ConfirmChan:   confirmCh,
				HeartBeatChan: heartBeatCh,
				DTContexts:    ctx,
			},
		},
	}

	// The Start method should recover from panic and not crash the test
	dm.Start()
}

func TestDTModule_InitWorker_Coverage(t *testing.T) {
	testCases := []struct {
		name          string
		expectedType  string
		expectedGroup string
		isDMI         bool // Special flag for DMI module
	}{
		{name: dtcommon.MemModule, expectedType: "dtmanager.MemWorker", expectedGroup: dtcommon.MemModule, isDMI: false},
		{name: dtcommon.TwinModule, expectedType: "dtmanager.TwinWorker", expectedGroup: dtcommon.TwinModule, isDMI: false},
		{name: dtcommon.DeviceModule, expectedType: "dtmanager.DeviceWorker", expectedGroup: dtcommon.DeviceModule, isDMI: false},
		{name: dtcommon.CommModule, expectedType: "dtmanager.CommWorker", expectedGroup: dtcommon.CommModule, isDMI: false},
		{name: dtcommon.DMIModule, expectedType: "dtmanager.DMIWorker", expectedGroup: dtcommon.DMIModule, isDMI: true},
	}

	ctx, err := dtcontext.InitDTContext()
	if err != nil {
		t.Fatalf("failed to init devicetwin context: %v", err)
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			recvCh := make(chan interface{})
			confirmCh := make(chan interface{})
			heartBeatCh := make(chan interface{})

			dm := &DTModule{
				Name: tt.name,
			}
			dm.InitWorker(recvCh, confirmCh, heartBeatCh, ctx)

			if dm.Worker == nil {
				t.Errorf("Worker not initialized for module %s", tt.name)
				return
			}

			// Check the type name
			actualType := fmt.Sprintf("%T", dm.Worker)
			if !strings.HasSuffix(actualType, tt.expectedType) {
				t.Errorf("Expected worker type %s, got %s", tt.expectedType, actualType)
			}

			// Access fields differently based on module type
			if tt.isDMI {
				// For DMI module, Worker field is embedded directly
				workerValue := reflect.ValueOf(dm.Worker)

				// Check the channels are properly set
				if workerValue.FieldByName("ReceiverChan").Interface() != recvCh {
					t.Errorf("ReceiverChan not properly set for module %s", tt.name)
				}
				if workerValue.FieldByName("ConfirmChan").Interface() != confirmCh {
					t.Errorf("ConfirmChan not properly set for module %s", tt.name)
				}
				if workerValue.FieldByName("HeartBeatChan").Interface() != heartBeatCh {
					t.Errorf("HeartBeatChan not properly set for module %s", tt.name)
				}
				if workerValue.FieldByName("DTContexts").Interface() != ctx {
					t.Errorf("DTContexts not properly set for module %s", tt.name)
				}

				// Check Group field
				if groupField := workerValue.FieldByName("Group"); groupField.IsValid() {
					if groupField.String() != tt.expectedGroup {
						t.Errorf("Expected group %s, got %s", tt.expectedGroup, groupField.String())
					}
				}
			} else {
				// For other modules, Worker field is nested
				workerValue := reflect.ValueOf(dm.Worker)
				workerField := workerValue.FieldByName("Worker")

				if !workerField.IsValid() {
					t.Errorf("Worker field not found in %T", dm.Worker)
					return
				}

				// Check the channels are properly set
				if workerField.FieldByName("ReceiverChan").Interface() != recvCh {
					t.Errorf("ReceiverChan not properly set for module %s", tt.name)
				}
				if workerField.FieldByName("ConfirmChan").Interface() != confirmCh {
					t.Errorf("ConfirmChan not properly set for module %s", tt.name)
				}
				if workerField.FieldByName("HeartBeatChan").Interface() != heartBeatCh {
					t.Errorf("HeartBeatChan not properly set for module %s", tt.name)
				}
				if workerField.FieldByName("DTContexts").Interface() != ctx {
					t.Errorf("DTContexts not properly set for module %s", tt.name)
				}

				// Check Group field
				if groupField := workerValue.FieldByName("Group"); groupField.IsValid() {
					if groupField.String() != tt.expectedGroup {
						t.Errorf("Expected group %s, got %s", tt.expectedGroup, groupField.String())
					}
				}
			}
		})
	}
}

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
		want   dtmanager.DTWorker
	}{
		{
			name: dtcommon.MemModule,
			fields: fields{
				Name: dtcommon.MemModule,
			},
			want: dtmanager.MemWorker{
				Worker: dtmanager.Worker{
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
			want: dtmanager.TwinWorker{
				Worker: dtmanager.Worker{
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
			want: dtmanager.DeviceWorker{
				Worker: dtmanager.Worker{
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
			want: dtmanager.CommWorker{
				Worker: dtmanager.Worker{
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
