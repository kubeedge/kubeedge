/*
Copyright 2023 The KubeEdge Authors.

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

package listwatchcacher

import (
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

type WatchHandlerManager interface {
	// ProcessWatch processes the watch request and returns an error if failed.
	ProcessWatch(app *metaserver.Application) error

	// FindAndAddNewWatchersNotExistInCloud iterate through all watch app in edge and
	// create new watchers if they don't exist in cloud watcherManager but should
	FindAndAddNewWatchersNotExistInCloud(
		nodeID string, msg model.Message,
		allWatchAppInEdge map[string]metaserver.Application,
		processFunc func(watchApp *metaserver.Application) error) error

	// FindAndRemoveTerminatedWatchersInEdge iterate through all watcher in cloud
	// watcherManager, and remove if they no longer exist in the edge
	FindAndRemoveTerminatedWatchersInEdge(
		allWatchAppInEdge map[string]metaserver.Application, nodeID string)
}

type watchHandlerManager struct {
	lock sync.RWMutex

	watcherManager *watcherManager

	messageLayer messagelayer.MessageLayer

	watchHandlers map[schema.GroupVersionResource]*WatchHandler
}

func NewWatchHandlerManager() WatchHandlerManager {
	return &watchHandlerManager{
		watcherManager: newWatcherManager(),
		messageLayer:   messagelayer.DynamicControllerMessageLayer(),
		watchHandlers:  make(map[schema.GroupVersionResource]*WatchHandler, 16),
	}
}

func (w *watchHandlerManager) ProcessWatch(app *metaserver.Application) error {
	gvr, _, _ := metaserver.ParseKey(app.Key)

	var err error
	var watchHandler *WatchHandler

	watchHandler = w.getHandler(gvr)
	if watchHandler == nil {
		if watchHandler, err = w.createHandler(gvr); err != nil {
			return err
		}
	}

	return watchHandler.Watch(app)
}

func (w *watchHandlerManager) FindAndAddNewWatchersNotExistInCloud(
	nodeID string, msg model.Message,
	allWatchAppInEdge map[string]metaserver.Application,
	processFunc func(watchApp *metaserver.Application) error) error {
	watchersInCloud := w.watcherManager.GetWatchersForNode(nodeID)

	newWatchApp := make([]metaserver.Application, 0)
	for ID, app := range allWatchAppInEdge {
		if _, exist := watchersInCloud[ID]; !exist {
			newWatchApp = append(newWatchApp, app)
			klog.Infof("added watch app %s", app.String())
		}
	}

	failedWatchApp := make(map[string]metaserver.Application)

	// create watcher for new added watch app
	for _, watchApp := range newWatchApp {
		err := processFunc(&watchApp)
		if err != nil {
			klog.Errorf("processWatchApp %s err: %v", watchApp.String(), err)

			failedWatchApp[watchApp.ID] = watchApp
			watchApp.Status = metaserver.Rejected

			apiErr, ok := err.(errors.APIStatus)
			if ok {
				watchApp.Error = errors.StatusError{ErrStatus: apiErr.Status()}
			} else {
				watchApp.Reason = err.Error()
			}
		}
	}

	respMsg := model.NewMessage(msg.GetID()).
		BuildRouter(modules.DynamicControllerModuleName, message.ResourceGroupName, msg.GetResource(), metaserver.ApplicationResp).
		FillBody(failedWatchApp)

	if err := w.messageLayer.Response(*respMsg); err != nil {
		klog.Warningf("send message error: %s, operation: %s, resource: %s", err, respMsg.GetOperation(), respMsg.GetResource())
		return err
	}

	return nil
}

func (w *watchHandlerManager) FindAndRemoveTerminatedWatchersInEdge(
	allWatchAppInEdge map[string]metaserver.Application, nodeID string) {
	watchersInCloud := w.watcherManager.GetWatchersForNode(nodeID)

	needRemovedWatchers := make([]*CacheWatcher, 0)
	for watcherID, watcher := range watchersInCloud {
		if _, exist := allWatchAppInEdge[watcherID]; !exist {
			needRemovedWatchers = append(needRemovedWatchers, watcher)
		}
	}

	// remove already terminated watcher
	for _, watcher := range needRemovedWatchers {
		klog.Infof("remove watcher(%s) for node %s", watcher.WatcherID, nodeID)
		w.watcherManager.DeleteWatcher(watcher)
	}
}

func (w *watchHandlerManager) getHandler(gvr schema.GroupVersionResource) *WatchHandler {
	w.lock.RLock()
	defer w.lock.RUnlock()

	return w.watchHandlers[gvr]
}

func (w *watchHandlerManager) createHandler(gvr schema.GroupVersionResource) (*WatchHandler, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	watchHandler, err := newWatchHandler(gvr, w.watcherManager)
	if err != nil {
		return nil, err
	}

	w.watchHandlers[gvr] = watchHandler

	return watchHandler, nil
}
