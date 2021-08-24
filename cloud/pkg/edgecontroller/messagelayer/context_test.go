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
	"reflect"
	"testing"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

const (
	thisModuleName     = "testcore"
	sendModuleName     = "testcore"
	receiveModuleName  = "testcore"
	responseModuleName = "testcore"
)

func init() {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	add := common.ModuleInfo{
		ModuleName: receiveModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(add)
	beehiveContext.AddModuleGroup(receiveModuleName, receiveModuleName)
}

func TestContextMessageLayer_Send_Receive_Response(t *testing.T) {
	type args struct {
		message model.Message
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"TestContextMessageLayer_Send_Receive_Response()",
			args{
				model.Message{
					Router: model.MessageRoute{
						Source: sendModuleName,
						Group:  receiveModuleName,
					},
					Content: "Hello Kubeedge",
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cml := &ContextMessageLayer{
				SendModuleName:     sendModuleName,
				ReceiveModuleName:  receiveModuleName,
				ResponseModuleName: responseModuleName,
			}
			if err := cml.Send(tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("ContextMessageLayer.Send() error = %v, wantErr %v", err, tt.wantErr)
			}

			got, err := cml.Receive()
			if err != nil {
				t.Errorf("ContextMessageLayer.Receive() failed. err: %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.args.message) {
				t.Errorf("ContextMessageLayer.Receive() failed. got: %v, want:%v", got, tt.args.message)
			}

			cml.Response(got)
		})
	}
}

func TestNewContextMessageLayer(t *testing.T) {
	config := &v1alpha1.ControllerContext{
		SendModule:     sendModuleName,
		ReceiveModule:  receiveModuleName,
		ResponseModule: responseModuleName,
	}

	tests := []struct {
		name string
	}{
		{
			"TestNewContextmessageLayer(): Case 1: create message layer",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewContextMessageLayer(config); got == nil {
				t.Errorf("NewContextMessageLayer() = %v", got)
			}
		})
	}
}
