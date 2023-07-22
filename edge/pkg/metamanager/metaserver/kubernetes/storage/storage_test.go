package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/request"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/agent"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	fakeclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/fake"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

func TestREST_PassThrough(t *testing.T) {
	type testCase struct {
		isConnectFailed            bool
		isSendSyncFailed           bool
		isLocalStored              bool
		isInsertLocalStorageFailed bool
	}
	cases := testCase{}

	fakeClient := fakeclient.Client{
		InsertOrUpdatePassThroughObjF: func(ctx context.Context, obj []byte, key string) error {
			if cases.isInsertLocalStorageFailed {
				return fmt.Errorf("insert local storage failed")
			}
			return nil
		},
		GetPassThroughObjF: func(ctx context.Context, key string) ([]byte, error) {
			if !cases.isLocalStored {
				return nil, fmt.Errorf("local does not store it")
			}
			return []byte("test"), nil
		},
	}
	patch := gomonkey.NewPatches()
	defer patch.Reset()
	patch.ApplyFunc(connect.IsConnected, func() bool {
		return !cases.isConnectFailed
	}).ApplyFunc(beehiveContext.SendSync, func(string, model.Message, time.Duration) (model.Message, error) {
		app := metaserver.Application{
			RespBody: []byte("test"),
			Status:   metaserver.Approved,
			Reason:   "ok",
		}
		if cases.isSendSyncFailed {
			app.Status = metaserver.Failed
			app.Reason = "isSendSyncFailed"
		}
		content, _ := json.Marshal(app)
		return model.Message{
			Content: content,
		}, nil
	}).ApplyGlobalVar(&imitator.DefaultV2Client, fakeClient)

	var tests = []struct {
		name    string
		rest    *REST
		info    request.RequestInfo
		cases   testCase
		want    []byte
		wantErr bool
	}{
		{
			name:    "test isConnectFailed ",
			info:    request.RequestInfo{},
			cases:   testCase{isConnectFailed: true},
			wantErr: true,
		}, {
			name:    "test isSendSyncFailed ",
			info:    request.RequestInfo{},
			cases:   testCase{isSendSyncFailed: true},
			wantErr: true,
		}, {
			name: "test get version from cloud failed, but local stored",
			info: request.RequestInfo{
				Path: "/versions",
				Verb: "get",
			},
			cases: testCase{isSendSyncFailed: true, isLocalStored: true},
			want:  []byte("test"),
		}, {
			name: "test successfully get the version from the cloud, but insert local storage failed ",
			info: request.RequestInfo{
				Path: "/versions",
				Verb: "get",
			},
			cases: testCase{isInsertLocalStorageFailed: true},
			want:  []byte("test"),
		}, {
			name: "test successfully get the version from the cloud ",
			info: request.RequestInfo{
				Path: "/versions",
				Verb: "get",
			},
			want: []byte("test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := request.WithRequestInfo(context.TODO(), &tt.info)
			rest := &REST{
				Agent: &agent.Agent{Applications: sync.Map{}},
			}
			cases = tt.cases
			got, err := rest.PassThrough(ctx, &metav1.GetOptions{})
			if (err != nil) != tt.wantErr {
				t.Errorf("PassThrough() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PassThrough() got = %v, want %v", got, tt.want)
			}
		})
	}
}
