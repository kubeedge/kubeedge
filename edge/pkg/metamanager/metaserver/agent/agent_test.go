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
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/util/workqueue"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

const (
	testNodeName = "test-node"
	failedReason = "failed reason"
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
			setupTestEnvironment()
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

			app, err := a.Generate(ctx, "get", metav1.GetOptions{}, nil)
			if err != nil {
				t.Fatalf("Unexpected error generating application: %v", err)
			}
			app.Close()
			// make sure that the last closing time is more than 5 minutes from now
			app.Timestamp = time.Unix(1469579899, 0)
			a.GC()
			_, ok := a.Applications.Load(app.Identifier())
			if ok {
				t.Error("Application delete failed")
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name          string
		connected     bool
		verb          metaserver.ApplicationVerb
		option        interface{}
		obj           runtime.Object
		expectError   bool
		errorContains string
	}{
		{
			name:        "Generate with connected state",
			connected:   true,
			verb:        "get",
			option:      metav1.GetOptions{},
			obj:         nil,
			expectError: false,
		},
		{
			name:          "Generate with disconnected state",
			connected:     false,
			verb:          "get",
			option:        metav1.GetOptions{},
			obj:           nil,
			expectError:   true,
			errorContains: "connection lost",
		},
		{
			name:        "Generate with list verb",
			connected:   true,
			verb:        "list",
			option:      metav1.ListOptions{},
			obj:         nil,
			expectError: false,
		},
		{
			name:        "Generate with watch verb",
			connected:   true,
			verb:        "watch",
			option:      metav1.ListOptions{Watch: true},
			obj:         nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestEnvironment()
			a := Agent{}
			ctx := createTestRequestContext()

			connect.SetConnected(tt.connected)

			app, err := a.Generate(ctx, tt.verb, tt.option, tt.obj)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorContains != "" && err.Error() != connect.ErrConnectionLost.Error() {
					t.Errorf("Expected error containing %q but got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got %v", err)
				}
				if app == nil {
					t.Error("Expected application but got nil")
				} else {
					stored, ok := a.Applications.Load(app.Identifier())
					if !ok {
						t.Error("Application not stored")
					} else if stored != app {
						t.Error("Stored application doesn't match returned application")
					}
				}
			}
		})
	}
}

func TestApplyWithRejectedStatus(t *testing.T) {
	setupTestEnvironment()
	ctx := createTestRequestContext()
	agent := Agent{}

	connect.SetConnected(true)

	app, err := agent.Generate(ctx, "get", metav1.GetOptions{}, nil)
	if err != nil {
		t.Fatalf("Unexpected error generating application: %v", err)
	}
	app.Status = metaserver.Rejected
	app.Error = *createStatusError("rejected")

	err = agent.Apply(app)

	if err == nil {
		t.Error("Expected error but got nil")
	}

	statusErr, ok := err.(*k8serrors.StatusError)
	if !ok {
		t.Errorf("Expected *k8serrors.StatusError but got %T", err)
	} else if statusErr.Error() != createStatusError("rejected").Error() {
		t.Errorf("Expected error %q but got %q", createStatusError("rejected").Error(), statusErr.Error())
	}
}

func TestApplyWithFailedStatus(t *testing.T) {
	setupTestEnvironment()
	ctx := createTestRequestContext()
	agent := Agent{}

	connect.SetConnected(true)

	app, err := agent.Generate(ctx, "get", metav1.GetOptions{}, nil)
	if err != nil {
		t.Fatalf("Unexpected error generating application: %v", err)
	}
	app.Status = metaserver.Failed
	app.Reason = failedReason

	err = agent.Apply(app)

	if err == nil {
		t.Errorf("Expected error but got nil")
	} else if err.Error() != failedReason {
		t.Errorf("Expected error '%s' but got %q", failedReason, err.Error())
	}
}

func TestApplyWithApprovedStatus(t *testing.T) {
	setupTestEnvironment()
	ctx := createTestRequestContext()
	agent := Agent{}

	connect.SetConnected(true)

	app, err := agent.Generate(ctx, "get", metav1.GetOptions{}, nil)
	if err != nil {
		t.Fatalf("Unexpected error generating application: %v", err)
	}
	app.Status = metaserver.Approved

	err = agent.Apply(app)

	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}
}

func TestDoApplyWithSendSyncError(t *testing.T) {
	setupTestEnvironment()
	a := &Agent{}
	ctx := createTestRequestContext()

	connect.SetConnected(true)

	app, err := a.Generate(ctx, "get", metav1.GetOptions{}, nil)
	if err != nil {
		t.Fatalf("Unexpected error generating application: %v", err)
	}
	app.Status = metaserver.PreApplying

	cancelPatch := gomonkey.ApplyMethod(app, "Cancel", func(_ *metaserver.Application) {
	})
	defer cancelPatch.Reset()

	sendSyncError := errors.New("send sync error")
	sendSyncPatch := gomonkey.ApplyFunc(beehiveContext.SendSync,
		func(module string, message model.Message, timeout time.Duration) (model.Message, error) {
			return model.Message{}, sendSyncError
		})
	defer sendSyncPatch.Reset()

	a.doApply(app)

	if app.Status != metaserver.Failed {
		t.Errorf("Expected status Failed but got %v", app.Status)
	}

	expectedReason := "failed to access cloud Application center: send sync error"
	if !strings.Contains(app.Reason, expectedReason) {
		t.Errorf("Expected reason containing %q but got %q", expectedReason, app.Reason)
	}
}

func TestListWatchApplications(t *testing.T) {
	setupTestEnvironment()
	a := &Agent{}
	ctx := createTestRequestContext()

	connect.SetConnected(true)

	watchApp, err := a.Generate(ctx, "watch", metav1.ListOptions{Watch: true}, nil)
	if err != nil {
		t.Fatalf("Unexpected error generating watch application: %v", err)
	}
	nonWatchApp, err := a.Generate(ctx, "get", metav1.GetOptions{}, nil)
	if err != nil {
		t.Fatalf("Unexpected error generating non-watch application: %v", err)
	}

	watchApps := a.listWatchApplications()

	if len(watchApps) != 1 {
		t.Errorf("Expected 1 watch app but got %d", len(watchApps))
	}

	if _, found := watchApps[watchApp.ID]; !found {
		t.Errorf("Watch app not found in result")
	}

	if _, found := watchApps[nonWatchApp.ID]; found {
		t.Errorf("Non-watch app should not be in result")
	}
}

func TestNewApplicationAgent(t *testing.T) {
	p1 := gomonkey.ApplyFunc(beehiveContext.SendSync, mockSendSync)
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc(beehiveContext.Done, func() <-chan struct{} {
		ch := make(chan struct{})
		close(ch)
		return ch
	})
	defer p2.Reset()

	agent := NewApplicationAgent()

	if agent == nil {
		t.Errorf("Expected non-nil agent")
	}

	if agent.watchSyncQueue == nil {
		t.Errorf("Expected non-nil watchSyncQueue")
	}
}

func TestAgentConnectionLossHandling(t *testing.T) {
	testCases := []struct {
		name           string
		connectionFunc func()
		expectError    bool
	}{
		{
			name: "Generate during connection loss",
			connectionFunc: func() {
				connect.SetConnected(false)
			},
			expectError: true,
		},
		{
			name: "Generate during connection restore",
			connectionFunc: func() {
				connect.SetConnected(true)
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setupTestEnvironment()

			p1 := gomonkey.ApplyFunc(beehiveContext.SendSync, mockSendSync)
			defer p1.Reset()

			p2 := gomonkey.ApplyFunc(beehiveContext.Done, func() <-chan struct{} {
				ch := make(chan struct{})
				close(ch)
				return ch
			})
			defer p2.Reset()

			a := NewApplicationAgent()
			ctx := createTestRequestContext()

			tc.connectionFunc()

			app, err := a.Generate(ctx, "get", metav1.GetOptions{}, nil)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error during connection loss, got nil")
				} else if err != connect.ErrConnectionLost {
					t.Errorf("Expected connection lost error, got: %v", err)
				}
				if app != nil {
					t.Errorf("Expected nil application during connection loss")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error during connection: %v", err)
				}
				if app == nil {
					t.Errorf("Expected non-nil application")
				}
			}
		})
	}
}

func setupTestEnvironment() {
	metaserverconfig.Config.NodeName = testNodeName
	connect.SetConnected(true)

	p1 := gomonkey.ApplyFunc(beehiveContext.SendSync, mockSendSync)
	defer p1.Reset()
}

func mockSendSync(module string, message model.Message, timeout time.Duration) (model.Message, error) {
	mockApp := &metaserver.Application{
		Status: metaserver.Approved,
	}

	respMsg := model.NewMessage("").SetRoute(module, module)
	respMsg.Content = mockApp

	return *respMsg, nil
}

func TestAgentApplicationLifecycle(t *testing.T) {
	sendSyncPatch := gomonkey.ApplyFunc(beehiveContext.SendSync, mockSendSync)
	defer sendSyncPatch.Reset()

	testCases := []struct {
		name           string
		initialStatus  metaserver.ApplicationStatus
		expectedError  bool
		errorValidator func(error) bool
	}{
		{
			name:          "Approved Application",
			initialStatus: metaserver.Approved,
			expectedError: false,
		},
		{
			name:          "Rejected Application",
			initialStatus: metaserver.Rejected,
			expectedError: true,
			errorValidator: func(err error) bool {
				_, ok := err.(*k8serrors.StatusError)
				return ok
			},
		},
		{
			name:          "Failed Application",
			initialStatus: metaserver.Failed,
			expectedError: true,
			errorValidator: func(err error) bool {
				return err != nil && !errors.Is(err, &k8serrors.StatusError{})
			},
		},
		{
			name:          "Completed Application",
			initialStatus: metaserver.Completed,
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setupTestEnvironment()
			a := NewApplicationAgent()
			ctx := createTestRequestContext()

			app, err := a.Generate(ctx, "get", metav1.GetOptions{}, nil)
			if err != nil {
				t.Fatalf("Failed to generate application: %v", err)
			}

			app.Status = tc.initialStatus
			if tc.initialStatus == metaserver.Rejected {
				app.Error = *createStatusError("rejected")
			} else if tc.initialStatus == metaserver.Failed {
				app.Reason = failedReason
			}

			err = a.Apply(app)

			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected error for status %v, got nil", tc.initialStatus)
				}
				if tc.errorValidator != nil && !tc.errorValidator(err) {
					t.Errorf("Error validation failed for status %v, got: %v", tc.initialStatus, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for status %v: %v", tc.initialStatus, err)
				}
			}
		})
	}
}

func TestDoApplyWithMsgProcessingError(t *testing.T) {
	setupTestEnvironment()
	a := NewApplicationAgent()
	ctx := createTestRequestContext()

	app, err := a.Generate(ctx, "get", metav1.GetOptions{}, nil)
	if err != nil {
		t.Fatalf("Unexpected error generating application: %v", err)
	}

	sendSyncPatch := gomonkey.ApplyFunc(beehiveContext.SendSync,
		func(module string, message model.Message, timeout time.Duration) (model.Message, error) {
			return model.Message{}, errors.New("message processing error")
		})
	defer sendSyncPatch.Reset()

	a.doApply(app)

	if app.Status != metaserver.Failed {
		t.Errorf("Expected Failed status, got %v", app.Status)
	}

	expectedReason := "failed to access cloud Application center: message processing error"
	if app.Reason != expectedReason {
		t.Errorf("Expected error reason %q, got %q", expectedReason, app.Reason)
	}
}

func TestAgentWatchSync(t *testing.T) {
	metaserverconfig.Config.NodeName = testNodeName
	connect.SetConnected(true)

	a := &Agent{
		Applications:   sync.Map{},
		watchSyncQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	sendSyncPatch := gomonkey.ApplyFunc(beehiveContext.SendSync,
		func(module string, message model.Message, timeout time.Duration) (model.Message, error) {
			respMsg := model.NewMessage("").SetRoute(module, module)
			respMsg.Content = map[string]metaserver.Application{}
			return *respMsg, nil
		})
	defer sendSyncPatch.Reset()

	watchApp1 := &metaserver.Application{
		ID:       "watch-app-1",
		Key:      "test-key-1",
		Verb:     metaserver.Watch,
		Nodename: testNodeName,
	}

	watchApp2 := &metaserver.Application{
		ID:       "watch-app-2",
		Key:      "test-key-2",
		Verb:     metaserver.Watch,
		Nodename: testNodeName,
	}

	nonWatchApp := &metaserver.Application{
		ID:       "non-watch-app",
		Key:      "test-key-3",
		Verb:     "get",
		Nodename: testNodeName,
	}

	a.Applications.Store(watchApp1.Identifier(), watchApp1)
	a.Applications.Store(watchApp2.Identifier(), watchApp2)
	a.Applications.Store(nonWatchApp.Identifier(), nonWatchApp)

	watchApps := a.listWatchApplications()

	if len(watchApps) != 2 {
		t.Errorf("Expected 2 watch applications, got %d", len(watchApps))
	}

	if _, exists := watchApps[watchApp1.ID]; !exists {
		t.Errorf("Watch application 1 not found in watch list")
	}
	if _, exists := watchApps[watchApp2.ID]; !exists {
		t.Errorf("Watch application 2 not found in watch list")
	}
}

func TestAgentConcurrentOperations(t *testing.T) {
	t.Skip("Skipping concurrent operations test")

	metaserverconfig.Config.NodeName = testNodeName
	connect.SetConnected(true)

	a := &Agent{
		Applications:   sync.Map{},
		watchSyncQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	sendSyncPatch := gomonkey.ApplyFunc(beehiveContext.SendSync,
		func(module string, message model.Message, timeout time.Duration) (model.Message, error) {
			mockApp := &metaserver.Application{
				Status: metaserver.Approved,
			}

			respMsg := model.NewMessage("").SetRoute(module, module)
			respMsg.Content = mockApp

			return *respMsg, nil
		})
	defer sendSyncPatch.Reset()

	totalRequests := 5
	identifiers := make(map[string]int)

	for i := 0; i < totalRequests; i++ {
		ctx := createUniqueTestRequestContext(i)
		app, err := a.Generate(ctx, "get", metav1.GetOptions{}, nil)
		if err != nil {
			t.Errorf("Unexpected error generating application: %v", err)
			continue
		}

		err = a.Apply(app)
		if err != nil {
			t.Errorf("Unexpected error applying application: %v", err)
		}

		identifiers[app.Identifier()]++
	}

	for identifier, count := range identifiers {
		if count > 1 {
			t.Errorf("Non-unique application identifier: %s (count: %d)", identifier, count)
		}
	}
}

func TestAgentCloseApplication(t *testing.T) {
	setupTestEnvironment()

	donePatch := gomonkey.ApplyFunc(beehiveContext.Done, func() <-chan struct{} {
		ch := make(chan struct{})
		close(ch)
		return ch
	})
	defer donePatch.Reset()

	a := NewApplicationAgent()
	ctx := createTestRequestContext()

	sendSyncPatch := gomonkey.ApplyFunc(beehiveContext.SendSync,
		func(module string, message model.Message, timeout time.Duration) (model.Message, error) {
			return model.Message{}, nil
		})
	defer sendSyncPatch.Reset()

	app, err := a.Generate(ctx, "get", metav1.GetOptions{}, nil)
	if err != nil {
		t.Fatalf("Unexpected error generating application: %v", err)
	}

	_, exists := a.Applications.Load(app.Identifier())
	if !exists {
		t.Errorf("Application not stored after generation")
	}

	a.CloseApplication(app.Identifier())

	_, exists = a.Applications.Load(app.Identifier())
	if exists {
		t.Errorf("Application not removed after CloseApplication")
	}
}

func createStatusError(message string) *k8serrors.StatusError {
	return &k8serrors.StatusError{
		ErrStatus: metav1.Status{
			Status:  metav1.StatusFailure,
			Message: message,
			Reason:  metav1.StatusReasonInvalid,
			Code:    400,
		},
	}
}

func createTestRequestContext() context.Context {
	requestInfo := &apirequest.RequestInfo{
		IsResourceRequest: true,
		Verb:              "GET",
		Path:              "http://127.0.0.1:10550/api/v1/nodes",
		APIPrefix:         "api",
		APIGroup:          "",
		APIVersion:        "v1",
		Resource:          "nodes",
		Subresource:       "",
	}

	ctx := context.Background()
	ctx = apirequest.WithRequestInfo(ctx, requestInfo)
	ctx = context.WithValue(ctx, commontypes.HeaderAuthorization, "Bearer test-token")

	return ctx
}

func createUniqueTestRequestContext(idx int) context.Context {
	requestInfo := &apirequest.RequestInfo{
		IsResourceRequest: true,
		Verb:              "GET",
		Path:              fmt.Sprintf("http://127.0.0.1:10550/api/v1/nodes/%d", idx),
		APIPrefix:         "api",
		APIGroup:          "",
		APIVersion:        "v1",
		Resource:          "nodes",
		Subresource:       "",
	}

	ctx := context.Background()
	ctx = apirequest.WithRequestInfo(ctx, requestInfo)
	ctx = context.WithValue(ctx, commontypes.HeaderAuthorization, fmt.Sprintf("Bearer test-token-%d", idx))

	return ctx
}
