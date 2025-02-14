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

package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"

	commontypes "github.com/kubeedge/kubeedge/common/types"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

func TestApplicationGC(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			"Test ApplicationGC Func",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metaserverconfig.Config.NodeName = "test"
			a := Agent{}
			requestInfo := &apirequest.RequestInfo{
				IsResourceRequest: true,
				Verb:              "GET",
				Path:              "http://127.0.0.1:10550/api/v1/nodes",
				APIPrefix:         "api",
				APIGroup:          "",
				APIVersion:        "v1",
				Resource:          "nodes",
			}
			ctx := apirequest.WithRequestInfo(context.Background(), requestInfo)
			ctx = context.WithValue(ctx, commontypes.HeaderAuthorization, "Bearer xxxx")

			connect.SetConnected(true)

			app, _ := a.Generate(ctx, "get", metav1.GetOptions{}, nil)
			app.Close()
			// make sure that the last closing time is more than 5 minutes from now
			app.Timestamp = time.Unix(1469579899, 0)
			a.GC()
			_, ok := a.Applications.Load(app.Identifier())
			if ok {
				t.Errorf("Application delete failed")
			}
		})
	}
}

func TestAgent_Apply(t *testing.T) {
	tests := []struct {
		name        string
		setupApp    func(*Agent) *metaserver.Application
		wantStatus  metaserver.ApplicationStatus
		wantErr     bool
		errContains string
	}{
		{
			name: "Unregistered application",
			setupApp: func(a *Agent) *metaserver.Application {
				return &metaserver.Application{ID: "unregistered"}
			},
			wantErr:     true,
			errContains: "has not been registered",
		},
		{
			name: "Approved application",
			setupApp: func(a *Agent) *metaserver.Application {
				app := &metaserver.Application{ID: "approved", Status: metaserver.Approved}
				a.Applications.Store(app.ID, app)
				return app
			},
			wantStatus: metaserver.Approved,
		},
		{
			name: "Failed application",
			setupApp: func(a *Agent) *metaserver.Application {
				app := &metaserver.Application{
					ID:     "failed",
					Status: metaserver.Failed,
					Reason: "test failure",
				}
				a.Applications.Store(app.ID, app)
				return app
			},
			wantErr:     true,
			errContains: "test failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewApplicationAgent()
			app := tt.setupApp(a)

			err := a.Apply(app)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if app.GetStatus() != tt.wantStatus {
				t.Errorf("got status %v, want %v", app.GetStatus(), tt.wantStatus)
			}
		})
	}
}

func TestAgent_ListWatchApplications(t *testing.T) {
	a := NewApplicationAgent()

	// Add some test applications
	watchApp := &metaserver.Application{
		ID:   "watch-app",
		Verb: metaserver.Watch,
	}
	nonWatchApp := &metaserver.Application{
		ID:   "non-watch-app",
		Verb: metaserver.Get,
	}

	a.Applications.Store(watchApp.ID, watchApp)
	a.Applications.Store(nonWatchApp.ID, nonWatchApp)

	watchApps := a.listWatchApplications()

	if len(watchApps) != 1 {
		t.Errorf("expected 1 watch application, got %d", len(watchApps))
	}

	if _, exists := watchApps[watchApp.ID]; !exists {
		t.Error("watch application not found in result")
	}

	if _, exists := watchApps[nonWatchApp.ID]; exists {
		t.Error("non-watch application found in result")
	}
}


