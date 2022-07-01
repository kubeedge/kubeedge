package statusmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/overridemanager"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/utils"
	appsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
)

type StatusManager interface {
	WatchStatus(utils.ResourceInfo) error
	CancelWatch(utils.ResourceInfo) error
	SetReconcileTriggerChan(chan event.GenericEvent)
	Start() error
}

type statusManager struct {
	ctx              context.Context
	mtx              sync.Mutex
	mgr              manager.Manager
	client           client.Client
	serializer       runtime.Serializer
	watching         map[schema.GroupVersionKind]context.CancelFunc
	watchCh          chan schema.GroupVersionKind
	cancelCh         chan schema.GroupVersionKind
	reconcileTrigger chan event.GenericEvent
	started          bool
}

func NewStatusManager(ctx context.Context, mgr manager.Manager, client client.Client, serializer runtime.Serializer) StatusManager {
	return &statusManager{
		ctx:        ctx,
		mtx:        sync.Mutex{},
		mgr:        mgr,
		client:     client,
		serializer: serializer,
		watching:   make(map[schema.GroupVersionKind]context.CancelFunc),
		watchCh:    make(chan schema.GroupVersionKind, 1024),
		cancelCh:   make(chan schema.GroupVersionKind, 1024),
	}
}

func (s *statusManager) WatchStatus(info utils.ResourceInfo) error {
	if !s.started {
		return fmt.Errorf("status manager has not started")
	}

	select {
	case s.watchCh <- infoToGVK(info):
	default:
		return fmt.Errorf("the wathCh of status manager is full, drop the info %s", info.String())
	}

	return nil
}

func (s *statusManager) CancelWatch(info utils.ResourceInfo) error {
	if !s.started {
		return fmt.Errorf("status manager has not started")
	}

	select {
	case s.cancelCh <- infoToGVK(info):
	default:
		return fmt.Errorf("the cancelCh of status manager is full, drop the info %s", info.String())
	}

	return nil
}

func (s *statusManager) Start() error {
	if s.reconcileTrigger == nil {
		return fmt.Errorf("reoncileTriger cannot be nil")
	}
	s.started = true
	go s.watchStatusWorker()
	go s.cancelWatchWorker()
	go s.waitForTerminatingWorkers()
	go wait.Until(s.watchControllersGC, 5*time.Minute, s.ctx.Done())
	return nil
}

func (s *statusManager) SetReconcileTriggerChan(ch chan event.GenericEvent) {
	s.reconcileTrigger = ch
}

func (s *statusManager) watchStatusWorker() {
	for gvk := range s.watchCh {
		if s.isWatching(gvk) {
			continue
		}
		ctx, cancel := context.WithCancel(s.ctx)
		s.markAsWatching(gvk, cancel)
		if err := s.startToWatch(ctx, gvk); err != nil {
			s.unmarkWatching(gvk)
			klog.Errorf("failed to start to watch status for gvk %s, %v", gvk, err)
			// TODO: if need retry
			continue
		}
		klog.V(4).Infof("start to watch status of gvk %s", gvk)
	}
	klog.Info("watchStatusWorker exited")
}

func (s *statusManager) cancelWatchWorker() {
	for info := range s.cancelCh {
		if !s.isWatching(info) {
			continue
		}
		// cancel the controller which is watching this kind of resource
		s.unmarkWatching(info)
	}
	// cancel all watching controllers
	s.mtx.Lock()
	defer s.mtx.Unlock()
	for _, cancel := range s.watching {
		if cancel != nil {
			cancel()
		}
	}
	klog.Info("cancelWatchWorker exited")
}

func (s *statusManager) waitForTerminatingWorkers() {
	<-s.ctx.Done()
	close(s.watchCh)
	close(s.cancelCh)
}

func (s *statusManager) isWatching(gvk schema.GroupVersionKind) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	_, ok := s.watching[gvk]
	return ok
}

func (s *statusManager) markAsWatching(gvk schema.GroupVersionKind, cancel context.CancelFunc) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.watching[gvk] = cancel
}

func (s *statusManager) unmarkWatching(gvk schema.GroupVersionKind) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if cancel := s.watching[gvk]; cancel != nil {
		cancel()
	}
	delete(s.watching, gvk)
}

func (s *statusManager) startToWatch(ctx context.Context, gvk schema.GroupVersionKind) error {
	controllerName := fmt.Sprintf("status-controller-for-%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
	controller, err := controller.NewUnmanaged(controllerName, s.mgr, controller.Options{
		Reconciler: &statusReconciler{
			Client:              s.client,
			GroupVersionKind:    gvk,
			Serializer:          s.serializer,
			Overrider:           &overridemanager.NameOverrider{},
			ReoncileTriggerChan: s.reconcileTrigger,
		},
	})
	if err != nil {
		klog.Errorf("failed to get new unmanaged controller for gvk %s, %v", gvk, err)
		return err
	}

	watchObj := &unstructured.Unstructured{}
	watchObj.SetGroupVersionKind(gvk)
	if err := controller.Watch(&source.Kind{Type: watchObj}, &handler.EnqueueRequestForOwner{
		OwnerType:    &appsv1alpha1.EdgeApplication{},
		IsController: true,
	}); err != nil {
		klog.Errorf("failed to add delete event watch to controller for gvk: %s, %v", gvk, err)
		return err
	}

	go func() {
		if err := controller.Start(ctx); err != nil {
			klog.Errorf("failed to start status controller for gvk %s, %v", gvk, err)
			return
		}
		klog.Infof("status controller stopped which was watching gvk %s", gvk)
	}()

	return nil
}

func (s *statusManager) watchControllersGC() {
	edgeAppList := &appsv1alpha1.EdgeApplicationList{}
	if err := s.client.List(s.ctx, edgeAppList); err != nil {
		klog.Errorf("failed to list EdgeApplication")
		return
	}

	infoMap := make(map[schema.GroupVersionKind]struct{})
	for _, edgeApp := range edgeAppList.Items {
		infos, err := utils.GetContainedResourceInfos(&edgeApp, s.serializer)
		if err != nil {
			klog.Errorf("failed to get resourceInfos from edgeApp %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
			continue
		}

		for _, info := range infos {
			infoMap[infoToGVK(info)] = struct{}{}
		}
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()
	for gvk := range s.watching {
		if _, ok := infoMap[gvk]; !ok {
			// no edgeapplication need to watch status of this gvk, so cancel watch of it
			if err := s.CancelWatch(utils.ResourceInfo{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind}); err != nil {
				klog.Errorf("statusControllersGC failed to cancel watch of gvk %s, %v", gvk, err)
				continue
			}
			klog.V(4).Infof("statusControllerGC cancel watch of gvk %s", gvk)
		}
	}
}

func infoToGVK(info utils.ResourceInfo) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   info.Group,
		Version: info.Version,
		Kind:    info.Kind,
	}
}
