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

package metamanager

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/mocks/beego"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	metaManagerConfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

const (
	MessageTest = "Test"
	// FailedDBOperation is common Database operation fail message
	FailedDBOperation = "Failed DB Operation"
	// ModuleNameEdged is name of edged module
	ModuleNameEdged = "edged"
	// ModuleNameEdgeHub is name of edgehub module
	ModuleNameEdgeHub = "websocket"
	// ModuleNameController is the name of the controller module
	ModuleNameController = "edgecontroller"
	// MarshalErroris common jsonMarshall error
	MarshalError = "Error to marshal message content: json: unsupported type: chan int"
	// OperationNodeConnection is message with operation publish
	OperationNodeConnection = "publish"
)

// errFailedDBOperation is common Database operation fail error
var errFailedDBOperation = errors.New(FailedDBOperation)

func init() {
	cfg := v1alpha1.NewDefaultEdgeCoreConfig()
	metaManagerConfig.InitConfigure(cfg.Modules.MetaManager)
}

// TestProcessInsert is function to test processInsert
func TestProcessInsert(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	dbm.DBAccess = ormerMock
	meta := newMetaManager(true)
	core.Register(meta)
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	beehiveContext.AddModule(meta.Name())
	beehiveContext.AddModuleGroup(meta.Name(), meta.Group())
	beehiveContext.AddModule(ModuleNameEdgeHub)
	beehiveContext.AddModuleGroup(ModuleNameEdgeHub, modules.HubGroup)
	beehiveContext.AddModule(ModuleNameEdged)

	//SaveMeta Failed, feedbackError SendToCloud
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), errFailedDBOperation).Times(1)
	msg := model.NewMessage("").BuildRouter(MetaManagerModuleName, GroupResource, model.ResourceTypePodStatus, model.InsertOperation)
	meta.processInsert(*msg)
	//beehiveContext.Send(MetaManagerModuleName, *msg)
	message, err := beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("EdgeHubChannelRegistration", func(t *testing.T) {
		if err != nil {
			t.Errorf("EdgeHub Channel not found: %v", err)
			return
		}
		want := "Error to save meta to DB: " + FailedDBOperation
		if message.GetContent() != want {
			t.Errorf("Wrong Error message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//SaveMeta Failed, feedbackError SendToEdged and 2 resources
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), errFailedDBOperation).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdged, GroupResource, model.ResourceTypePodStatus+"/secondRes", model.InsertOperation)
	meta.processInsert(*msg)
	message, err = beehiveContext.Receive(ModuleNameEdged)
	t.Run("ErrorMessageToEdged", func(t *testing.T) {
		if err != nil {
			t.Errorf("EdgeD Channel not found: %v", err)
			return
		}
		want := "Error to save meta to DB: " + FailedDBOperation
		if message.GetContent() != want {
			t.Errorf("Wrong Error message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//jsonMarshall fail
	msg = model.NewMessage("").BuildRouter(ModuleNameEdged, GroupResource, model.ResourceTypePodStatus, model.InsertOperation).FillBody(make(chan int))
	meta.processInsert(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("MarshallFail", func(t *testing.T) {
		want := MarshalError
		if message.GetContent() != want {
			t.Errorf("Wrong Error message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Succesful Case and 3 resources
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdged, GroupResource, model.ResourceTypePodStatus+"/secondRes"+"/thirdRes", model.InsertOperation)
	meta.processInsert(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("InsertMessageToEdged", func(t *testing.T) {
		want := model.InsertOperation
		if message.GetOperation() != want {
			t.Errorf("Wrong message received : Wanted %v and Got %v", want, message.GetOperation())
		}
	})
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("ResponseMessageToEdgeHub", func(t *testing.T) {
		want := OK
		if message.GetContent() != want {
			t.Errorf("Wrong message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})
}

// TestProcessUpdate is function to test processUpdate
func TestProcessUpdate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	rawSeterMock := beego.NewMockRawSeter(mockCtrl)
	querySeterMock := beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock
	meta := newMetaManager(true)
	core.Register(meta)
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	beehiveContext.AddModule(meta.Name())
	beehiveContext.AddModuleGroup(meta.Name(), meta.Group())
	beehiveContext.AddModule(ModuleNameEdgeHub)
	beehiveContext.AddModuleGroup(ModuleNameEdgeHub, modules.HubGroup)
	beehiveContext.AddModule(EdgeFunctionModel)
	beehiveContext.AddModule(ModuleNameEdged)

	//jsonMarshall fail
	msg := model.NewMessage("").BuildRouter(ModuleNameEdged, GroupResource, model.ResourceTypePodStatus, model.UpdateOperation).FillBody(make(chan int))
	meta.processUpdate(*msg)
	message, _ := beehiveContext.Receive(ModuleNameEdged)
	t.Run("MarshallFail", func(t *testing.T) {
		want := MarshalError
		if message.GetContent() != want {
			t.Errorf("Wrong Error message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Database save error
	rawSeterMock.EXPECT().Exec().Return(nil, errFailedDBOperation).Times(1)
	ormerMock.EXPECT().Raw(gomock.Any(), gomock.Any()).Return(rawSeterMock).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdged, GroupResource, model.ResourceTypePodStatus, model.UpdateOperation)
	meta.processUpdate(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("DatabaseSaveError", func(t *testing.T) {
		want := "Error to update meta to DB: " + FailedDBOperation
		if message.GetContent() != want {
			t.Errorf("Wrong Error message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//resourceUnchanged true
	fakeDao := new([]dao.Meta)
	fakeDaoArray := make([]dao.Meta, 1)
	fakeDaoArray[0] = dao.Meta{Key: "Test", Value: "\"test\""}
	fakeDao = &fakeDaoArray
	querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDao).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdged, GroupResource, "test/"+model.ResourceTypePodStatus, model.UpdateOperation).FillBody("test")
	meta.processUpdate(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("ResourceUnchangedTrue", func(t *testing.T) {
		want := OK
		if message.GetContent() != want {
			t.Errorf("Resource Unchanged Case Failed: Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Success Case Source Edged, sync = true
	ormerMock.EXPECT().Raw(gomock.Any(), gomock.Any()).Return(rawSeterMock).Times(1)
	rawSeterMock.EXPECT().Exec().Return(nil, nil).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdged, GroupResource, model.ResourceTypePodStatus, model.UpdateOperation)
	meta.processUpdate(*msg)
	edgehubMsg, _ := beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("SuccessSourceEdgedReceiveEdgehub", func(t *testing.T) {
		want := model.UpdateOperation
		if edgehubMsg.GetOperation() != want {
			t.Errorf("Wrong message received : Wanted operation %v and Got operation %v", want, edgehubMsg.GetOperation())
		}
	})
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("SuccessSourceEdgedReceiveEdged", func(t *testing.T) {
		want := OK
		if message.GetContent() != want {
			t.Errorf("Wrong message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Success Case Source CloudControlerModel
	ormerMock.EXPECT().Raw(gomock.Any(), gomock.Any()).Return(rawSeterMock).Times(1)
	rawSeterMock.EXPECT().Exec().Return(nil, nil).Times(1)
	msg = model.NewMessage("").BuildRouter(CloudControlerModel, GroupResource, model.ResourceTypePodStatus, model.UpdateOperation)
	meta.processUpdate(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("SuccessSend[CloudController->Edged]", func(t *testing.T) {
		want := CloudControlerModel
		if message.GetSource() != want {
			t.Errorf("Wrong message received : Wanted from source %v and Got from source %v", want, message.GetSource())
		}
	})
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("SuccessSendCloud[CloudController->EdgeHub]", func(t *testing.T) {
		want := OK
		if message.GetContent() != want {
			t.Errorf("Wrong message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Success Case Source CloudFunctionModel
	ormerMock.EXPECT().Raw(gomock.Any(), gomock.Any()).Return(rawSeterMock).Times(1)
	rawSeterMock.EXPECT().Exec().Return(nil, nil).Times(1)
	msg = model.NewMessage("").BuildRouter(CloudFunctionModel, GroupResource, model.ResourceTypePodStatus, model.UpdateOperation)
	meta.processUpdate(*msg)
	message, _ = beehiveContext.Receive(EdgeFunctionModel)
	t.Run("SuccessSend[CloudFunction->EdgeFunction]", func(t *testing.T) {
		want := CloudFunctionModel
		if message.GetSource() != want {
			t.Errorf("Wrong message received : Wanted from source %v and Got from source %v", want, message.GetSource())
		}
	})

	//Success Case Source EdgeFunctionModel
	rawSeterMock.EXPECT().Exec().Return(nil, nil).Times(1)
	ormerMock.EXPECT().Raw(gomock.Any(), gomock.Any()).Return(rawSeterMock).Times(1)
	//rawSeterMock.EXPECT().Exec().Return(nil, nil).Times(1)
	msg = model.NewMessage("").BuildRouter(EdgeFunctionModel, GroupResource, model.ResourceTypePodStatus, model.UpdateOperation)
	meta.processUpdate(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("SuccessSend[EdgeFunction->EdgeHub]", func(t *testing.T) {
		want := EdgeFunctionModel
		if message.GetSource() != want {
			t.Errorf("Wrong message received : Wanted from source %v and Got from source %v", want, message.GetSource())
		}
	})
}

// TestProcessResponse is function to test processResponse
func TestProcessResponse(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	rawSeterMock := beego.NewMockRawSeter(mockCtrl)
	dbm.DBAccess = ormerMock
	meta := newMetaManager(true)
	core.Register(meta)
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	beehiveContext.AddModule(meta.Name())
	beehiveContext.AddModuleGroup(meta.Name(), meta.Group())
	beehiveContext.AddModule(ModuleNameEdgeHub)
	beehiveContext.AddModuleGroup(ModuleNameEdgeHub, modules.HubGroup)
	beehiveContext.AddModule(EdgeFunctionModel)
	beehiveContext.AddModule(ModuleNameEdged)

	//jsonMarshall fail
	msg := model.NewMessage("").BuildRouter(ModuleNameEdged, GroupResource, model.ResourceTypePodStatus, model.ResponseOperation).FillBody(make(chan int))
	beehiveContext.Send(MetaManagerModuleName, *msg)
	meta.processResponse(*msg)
	message, _ := beehiveContext.Receive(ModuleNameEdged)
	t.Run("MarshallFail", func(t *testing.T) {
		want := MarshalError
		if message.GetContent() != want {
			t.Errorf("Wrong Error message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Database save error
	ormerMock.EXPECT().Raw(gomock.Any(), gomock.Any()).Return(rawSeterMock).Times(1)
	rawSeterMock.EXPECT().Exec().Return(nil, errFailedDBOperation).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdged, GroupResource, model.ResourceTypePodStatus, model.ResponseOperation)
	meta.processResponse(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("DatabaseSaveError", func(t *testing.T) {
		want := "Error to update meta to DB: " + FailedDBOperation
		if message.GetContent() != want {
			t.Errorf("Wrong Error message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Success Case Source EdgeD
	ormerMock.EXPECT().Raw(gomock.Any(), gomock.Any()).Return(rawSeterMock).Times(1)
	rawSeterMock.EXPECT().Exec().Return(nil, nil).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdged, GroupResource, model.ResourceTypePodStatus, model.ResponseOperation)
	meta.processResponse(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("SuccessSourceEdged", func(t *testing.T) {
		want := ModuleNameEdged
		if message.GetSource() != want {
			t.Errorf("Wrong message received : Wanted from source %v and Got from source %v", want, message.GetSource())
		}
	})

	//Success Case Source EdgeHub
	ormerMock.EXPECT().Raw(gomock.Any(), gomock.Any()).Return(rawSeterMock).Times(1)
	rawSeterMock.EXPECT().Exec().Return(nil, nil).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameController, GroupResource, model.ResourceTypePodStatus, model.ResponseOperation)
	meta.processResponse(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("SuccessSourceEdgeHub", func(t *testing.T) {
		want := ModuleNameController
		if message.GetSource() != want {
			t.Errorf("Wrong message received : Wanted from source %v and Got from source %v", want, message.GetSource())
		}
	})
}

// TestProcessDelete is function to test processDelete
func TestProcessDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	querySeterMock := beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock
	meta := newMetaManager(true)
	core.Register(meta)
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	beehiveContext.AddModule(meta.Name())
	beehiveContext.AddModuleGroup(meta.Name(), meta.Group())
	beehiveContext.AddModule(ModuleNameEdgeHub)
	beehiveContext.AddModuleGroup(ModuleNameEdgeHub, modules.HubGroup)
	beehiveContext.AddModule(ModuleNameEdged)

	//Database Save Error
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), errFailedDBOperation).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	msg := model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, model.ResourceTypePodStatus, model.DeleteOperation)
	meta.processDelete(*msg)
	message, _ := beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("DatabaseDeleteError", func(t *testing.T) {
		want := "Error to delete meta to DB: " + FailedDBOperation
		if message.GetContent() != want {
			t.Errorf("Wrong message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Success Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, model.ResourceTypePodStatus, model.DeleteOperation)
	meta.processDelete(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("SuccessSourceEdgeHub", func(t *testing.T) {
		want := ModuleNameEdgeHub
		if message.GetSource() != want {
			t.Errorf("Wrong message received : Wanted from source %v and Got from source %v", want, message.GetSource())
		}
	})
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("SuccessResponseOK", func(t *testing.T) {
		want := OK
		if message.GetContent() != want {
			t.Errorf("Wrong message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})
}

// TestProcessQuery is function to test processQuery
func TestProcessQuery(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	querySeterMock := beego.NewMockQuerySeter(mockCtrl)
	rawSeterMock := beego.NewMockRawSeter(mockCtrl)
	dbm.DBAccess = ormerMock
	meta := newMetaManager(true)
	core.Register(meta)
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	beehiveContext.AddModule(meta.Name())
	beehiveContext.AddModuleGroup(meta.Name(), meta.Group())
	beehiveContext.AddModule(ModuleNameEdgeHub)
	beehiveContext.AddModuleGroup(ModuleNameEdgeHub, modules.HubGroup)
	beehiveContext.AddModule(ModuleNameEdged)

	//process remote query sync error case
	msg := model.NewMessage("").BuildRouter(ModuleNameEdged, GroupResource, model.ResourceTypePodStatus, OperationNodeConnection).FillBody(connect.CloudConnected)
	meta.processNodeConnection(*msg)
	//wait for message to be received by metaManager and get processed
	time.Sleep(1 * time.Second)
	t.Run("ConnectedTrue", func(t *testing.T) {
		if metaManagerConfig.Connected != true {
			t.Errorf("Connected was not set to true")
		}
	})

	//process remote query jsonMarshall error
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), errFailedDBOperation).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, "test/"+model.ResourceTypeConfigmap, model.QueryOperation)
	meta.processQuery(*msg)
	message, _ := beehiveContext.Receive(ModuleNameEdgeHub)
	msg = model.NewMessage(message.GetID()).BuildRouter(ModuleNameEdgeHub, GroupResource, "test/"+model.ResourceTypeConfigmap, model.QueryOperation).FillBody(make(chan int))
	beehiveContext.SendResp(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("ProcessRemoteQueryMarshallFail", func(t *testing.T) {
		want := MarshalError
		if message.GetContent() != want {
			t.Errorf("Wrong Error message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//process remote query db fail
	rawSeterMock.EXPECT().Exec().Return(nil, errFailedDBOperation).Times(1)
	ormerMock.EXPECT().Raw(gomock.Any(), gomock.Any()).Return(rawSeterMock).Times(1)
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), errFailedDBOperation).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, "test/"+model.ResourceTypeConfigmap, model.QueryOperation)
	meta.processQuery(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	msg = model.NewMessage(message.GetID()).BuildRouter(ModuleNameEdged, GroupResource, "test/"+model.ResourceTypeConfigmap, model.QueryOperation).FillBody("TestMessage")
	beehiveContext.SendResp(*msg)
	beehiveContext.Receive(ModuleNameEdgeHub)
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("ProcessRemoteQueryDbFail", func(t *testing.T) {
		want := "TestMessage"
		if message.GetContent() != want {
			t.Errorf("Wrong message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//No error and connected true
	fakeDao := new([]dao.Meta)
	fakeDaoArray := make([]dao.Meta, 1)
	fakeDaoArray[0] = dao.Meta{Key: "Test", Value: MessageTest}
	fakeDao = &fakeDaoArray
	querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDao).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, "test/"+model.ResourceTypeConfigmap, model.QueryOperation)
	meta.processQuery(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("DatabaseNoErrorAndMetaFound", func(t *testing.T) {
		want := make([]string, 1)
		want[0] = MessageTest
		bytesWant, _ := json.Marshal(want)
		bytesGot, _ := json.Marshal(message.GetContent())
		if string(bytesGot) != string(bytesWant) {
			t.Errorf("Wrong message receive : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//ResId Nil database error
	msg = model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, model.ResourceTypePodStatus, OperationNodeConnection).FillBody(connect.CloudDisconnected)

	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), errFailedDBOperation).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, model.ResourceTypeConfigmap, model.QueryOperation)
	meta.processQuery(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("ResIDNilDatabaseError", func(t *testing.T) {
		want := "Error to query meta in DB: " + FailedDBOperation
		if message.GetContent() != want {
			t.Errorf("Wrong message receive : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//ResID not nil database error
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), errFailedDBOperation).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, "test/test/"+model.ResourceTypeConfigmap, model.QueryOperation)
	//beehiveContext.Send(MetaManagerModuleName, *msg)
	meta.processQuery(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("ResIDNotNilDatabaseError", func(t *testing.T) {
		want := "Error to query meta in DB: " + FailedDBOperation
		if message.GetContent() != want {
			t.Errorf("Wrong message receive : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//ResID not nil Success Case
	querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDao).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, "test/test/"+model.ResourceTypeConfigmap, model.QueryOperation)
	//beehiveContext.Send(MetaManagerModuleName, *msg)
	meta.processQuery(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdged)
	t.Run("DatabaseNoErrorAndMetaFound", func(t *testing.T) {
		want := make([]string, 1)
		want[0] = MessageTest
		bytesWant, _ := json.Marshal(want)
		bytesGot, _ := json.Marshal(message.GetContent())
		if string(bytesGot) != string(bytesWant) {
			t.Errorf("Wrong message receive : Wanted %v and Got %v", want, message.GetContent())
		}
	})
}

// TestProcessNodeConnection is function to test processNodeConnection
func TestProcessNodeConnection(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	dbm.DBAccess = ormerMock
	meta := newMetaManager(true)
	core.Register(meta)
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	beehiveContext.AddModule(meta.Name())
	beehiveContext.AddModuleGroup(meta.Name(), meta.Group())
	beehiveContext.AddModule(ModuleNameEdgeHub)
	beehiveContext.AddModule(EdgeFunctionModel)

	//connected true
	msg := model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, model.ResourceTypePodStatus, OperationNodeConnection).FillBody(connect.CloudConnected)
	meta.processNodeConnection(*msg)
	//wait for message to be received by metaManager and get processed
	time.Sleep(1 * time.Second)
	t.Run("ConnectedTrue", func(t *testing.T) {
		if metaManagerConfig.Connected != true {
			t.Errorf("Connected was not set to true")
		}
	})

	//connected false
	msg = model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, model.ResourceTypePodStatus, OperationNodeConnection).FillBody(connect.CloudDisconnected)
	meta.processNodeConnection(*msg)
	//wait for message to be received by metaManager and get processed
	time.Sleep(1 * time.Second)
	t.Run("ConnectedFalse", func(t *testing.T) {
		if metaManagerConfig.Connected != false {
			t.Errorf("Connected was not set to false")
		}
	})
}

// TestProcessSync is function to test processSync
func TestProcessSync(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	querySeterMock := beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock
	meta := newMetaManager(true)
	core.Register(meta)
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	beehiveContext.AddModule(meta.Name())
	beehiveContext.AddModuleGroup(meta.Name(), meta.Group())
	beehiveContext.AddModule(ModuleNameEdgeHub)
	beehiveContext.AddModuleGroup(ModuleNameEdgeHub, modules.HubGroup)

	//QueryMeta Length > 0 Success Case
	fakeDao := new([]dao.Meta)
	fakeDaoArray := make([]dao.Meta, 1)
	fakeDaoArray[0] = dao.Meta{Key: "Test/Test/Test", Value: "\"Test\""}
	fakeDao = &fakeDaoArray
	querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDao).Return(int64(1), nil).Times(2)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(2)
	meta.processSync()
	message, _ := beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("QueryMetaLengthSuccess Case", func(t *testing.T) {
		want := make([]interface{}, 1)
		want[0] = "Test"
		bytesWant, _ := json.Marshal(want)
		bytesGot, _ := json.Marshal(message.GetContent())
		if string(bytesGot) != string(bytesWant) {
			t.Errorf("Wrong message receive : Wanted %v and Got %v", want, message.GetContent())
		}
	})
}

// TestProcessFunctionAction is function to test processFunctionAction
func TestProcessFunctionAction(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	dbm.DBAccess = ormerMock
	meta := newMetaManager(true)
	core.Register(meta)
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	beehiveContext.AddModule(meta.Name())
	beehiveContext.AddModuleGroup(meta.Name(), meta.Group())
	beehiveContext.AddModule(ModuleNameEdgeHub)
	beehiveContext.AddModuleGroup(ModuleNameEdgeHub, modules.HubGroup)
	beehiveContext.AddModule(EdgeFunctionModel)

	//jsonMarshall fail
	msg := model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, model.ResourceTypePodStatus, OperationFunctionAction).FillBody(make(chan int))
	meta.processFunctionAction(*msg)
	//beehiveContext.Send(MetaManagerModuleName, *msg)
	message, _ := beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("MarshallFail", func(t *testing.T) {
		want := MarshalError
		if message.GetContent() != want {
			t.Errorf("Wrong Error message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Database Save Error
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), errFailedDBOperation).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, model.ResourceTypePodStatus, OperationFunctionAction).FillBody("test")
	meta.processFunctionAction(*msg)
	//beehiveContext.Send(MetaManagerModuleName, *msg)
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("DatabaseSaveError", func(t *testing.T) {
		want := "Error to save meta to DB: " + FailedDBOperation
		if message.GetContent() != want {
			t.Errorf("Wrong message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Success Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	msg = model.NewMessage("").BuildRouter(ModuleNameEdgeHub, GroupResource, model.ResourceTypePodStatus, OperationFunctionAction)
	meta.processFunctionAction(*msg)
	//beehiveContext.Send(MetaManagerModuleName, *msg)
	message, _ = beehiveContext.Receive(EdgeFunctionModel)
	t.Run("SuccessCase", func(t *testing.T) {
		want := ModuleNameEdgeHub
		if message.GetSource() != want {
			t.Errorf("Wrong message received : Wanted from source %v and Got from source %v", want, message.GetSource())
		}
	})
}

// TestProcessFunctionActionResult is function to test processFunctionActionResult
func TestProcessFunctionActionResult(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	dbm.DBAccess = ormerMock
	meta := newMetaManager(true)
	core.Register(meta)
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	beehiveContext.AddModule(meta.Name())
	beehiveContext.AddModuleGroup(meta.Name(), meta.Group())
	beehiveContext.AddModule(ModuleNameEdgeHub)
	beehiveContext.AddModuleGroup(ModuleNameEdgeHub, modules.HubGroup)

	//jsonMarshall fail
	msg := model.NewMessage("").BuildRouter(EdgeFunctionModel, GroupResource, model.ResourceTypePodStatus, OperationFunctionActionResult).FillBody(make(chan int))
	meta.processFunctionActionResult(*msg)
	message, _ := beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("MarshallFail", func(t *testing.T) {
		want := MarshalError
		if message.GetContent() != want {
			t.Errorf("Wrong Error message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Database Save Error
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), errFailedDBOperation).Times(1)
	msg = model.NewMessage("").BuildRouter(EdgeFunctionModel, GroupResource, model.ResourceTypePodStatus, OperationFunctionActionResult)
	meta.processFunctionActionResult(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("DatabaseSaveError", func(t *testing.T) {
		want := "Error to save meta to DB: " + FailedDBOperation
		if message.GetContent() != want {
			t.Errorf("Wrong message received : Wanted %v and Got %v", want, message.GetContent())
		}
	})

	//Success Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	msg = model.NewMessage("").BuildRouter(EdgeFunctionModel, GroupResource, model.ResourceTypePodStatus, OperationFunctionActionResult)
	meta.processFunctionActionResult(*msg)
	message, _ = beehiveContext.Receive(ModuleNameEdgeHub)
	t.Run("SuccessCase", func(t *testing.T) {
		want := EdgeFunctionModel
		if message.GetSource() != want {
			t.Errorf("Wrong message received : Wanted %v and Got %v", want, message.GetSource())
		}
	})
}
