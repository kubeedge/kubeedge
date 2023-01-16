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
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	edgemodule "github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

var DefaultAgent = NewApplicationAgent()

// Agent used for generating application and do apply
type Agent struct {
	Applications sync.Map //store struct application
	// watchSyncQueue store the watch request sync message
	watchSyncQueue workqueue.RateLimitingInterface
}

// NewApplicationAgent create edge agent for list/watch
func NewApplicationAgent() *Agent {
	defaultAgent := &Agent{
		watchSyncQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	go wait.Until(func() {
		defaultAgent.GC()
	}, time.Minute*5, beehiveContext.Done())

	go defaultAgent.watchSync()

	return defaultAgent
}

func (a *Agent) Generate(ctx context.Context, verb metaserver.ApplicationVerb, option interface{}, obj runtime.Object) (*metaserver.Application, error) {
	// If the connection is lost between EdgeCore and CloudCore, we do not generate
	// the application since we can not send the application to the CloudCore
	if !connect.IsConnected() {
		return nil, connect.ErrConnectionLost
	}

	key, err := metaserver.KeyFuncReq(ctx, "")
	if err != nil {
		return nil, err
	}

	info, _ := apirequest.RequestInfoFrom(ctx)

	app, err := metaserver.NewApplication(ctx, key, verb, metaserverconfig.Config.NodeName, info.Subresource, option, obj)
	if err != nil {
		return nil, err
	}
	store, ok := a.Applications.LoadOrStore(app.Identifier(), app)
	if ok {
		app = store.(*metaserver.Application)
		app.Add()
		return app, nil
	}
	return app, nil
}

func (a *Agent) Apply(app *metaserver.Application) error {
	store, ok := a.Applications.Load(app.Identifier())
	if !ok {
		return fmt.Errorf("application %v has not been registered to agent", app.String())
	}
	app = store.(*metaserver.Application)
	switch app.GetStatus() {
	case metaserver.PreApplying:
		go a.doApply(app)
	case metaserver.Completed:
		app.Reset()
		go a.doApply(app)
	case metaserver.Rejected:
		return &app.Error
	case metaserver.Failed:
		return errors.New(app.Reason)
	case metaserver.Approved:
		return nil
	case metaserver.InApplying:
		//continue
	}
	app.Wait()
	if app.GetStatus() == metaserver.Rejected {
		return &app.Error
	}
	if app.GetStatus() != metaserver.Approved {
		return errors.New(app.Reason)
	}
	return nil
}

func (a *Agent) doApply(app *metaserver.Application) {
	defer app.Cancel()
	// encapsulate as a message
	app.Status = metaserver.InApplying
	msg := model.NewMessage("").SetRoute(metaserver.MetaServerSource, modules.DynamicControllerModuleGroup).FillBody(app)
	msg.SetResourceOperation("null", "null")
	resp, err := beehiveContext.SendSync(edgemodule.EdgeHubModuleName, *msg, 10*time.Second)
	if err != nil {
		app.Status = metaserver.Failed
		app.Reason = fmt.Sprintf("failed to access cloud Application center: %v", err)
		return
	}

	retApp, err := metaserver.MsgToApplication(resp)
	if err != nil {
		app.Status = metaserver.Failed
		app.Reason = fmt.Sprintf("failed to get Application from resp msg: %v", err)
		return
	}

	//merge returned application to local application
	app.Status = retApp.Status
	app.Reason = retApp.Reason
	app.Error = retApp.Error
	app.RespBody = retApp.RespBody
}

func (a *Agent) CloseApplication(appID string) {
	a.Applications.Delete(appID)
	a.SyncWatchAppOnConnected()
}

func (a *Agent) GC() {
	a.Applications.Range(func(key, value interface{}) bool {
		app := value.(*metaserver.Application)
		lastCloseTime := app.LastCloseTime()
		if !lastCloseTime.IsZero() && time.Since(lastCloseTime) >= time.Minute*5 {
			a.Applications.Delete(key)
		}
		return true
	})
}

func (a *Agent) SyncWatchAppOnConnected() {
	a.watchSyncQueue.Add("SyncWatchApp")
}

func (a *Agent) watchSync() {
	for a.processNextWorkItem() {
	}
}

func (a *Agent) processNextWorkItem() bool {
	key, quit := a.watchSyncQueue.Get()
	if quit {
		return false
	}
	defer a.watchSyncQueue.Done(key)

	err := a.syncWatchApplications()
	if err == nil {
		a.watchSyncQueue.Forget(key)
		return true
	}

	a.watchSyncQueue.AddRateLimited(key)
	return true
}

func (a *Agent) syncWatchApplications() error {
	watchApps := a.listWatchApplications()

	msg := model.NewMessage("").SetRoute(metaserver.MetaServerSource, modules.DynamicControllerModuleGroup).FillBody(watchApps)
	msg.SetResourceOperation(metaserver.WatchAppSync, "null")

	resp, err := beehiveContext.SendSync(edgemodule.EdgeHubModuleName, *msg, 10*time.Second)
	if err != nil {
		klog.Errorf("syncWatchApplications SendSync msg err: %v", err)
		return err
	}

	failedWatchApps, err := metaserver.MsgToApplications(resp)
	if err != nil {
		klog.Errorf("syncWatchApplications MsgToApplications err: %v", err)
		return err
	}

	klog.Errorf("failed to process watch apps: %+v", failedWatchApps)

	return nil
}

func (a *Agent) listWatchApplications() map[string]metaserver.Application {
	watchApps := make(map[string]metaserver.Application)
	a.Applications.Range(func(key, value interface{}) bool {
		app := value.(*metaserver.Application)
		if app.Verb != metaserver.Watch {
			return true
		}

		watchApps[app.ID] = *app
		return true
	})

	return watchApps
}
