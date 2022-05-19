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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	groupingv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/grouping/v1alpha1"
)

type StatusManager interface {
	WatchStatus(ResourceInfo) error
	CancelWatch(ResourceInfo) error
	Start()
}

type ResourceInfo struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

func (c *ResourceInfo) String() string {
	return fmt.Sprintf("%s/%s, kind=%s, namespace=%s, name=%s", c.Group, c.Version, c.Kind, c.Namespace, c.Name)
}

type statusManager struct {
	ctx        context.Context
	mtx        sync.Mutex
	mgr        manager.Manager
	client     client.Client
	serializer runtime.Serializer
	watching   map[schema.GroupVersionKind]context.CancelFunc
	watchCh    chan ResourceInfo
	cancelCh   chan ResourceInfo
	started    bool
}

func NewStatusManager(ctx context.Context, mgr manager.Manager, client client.Client, serializer runtime.Serializer) StatusManager {
	return &statusManager{
		ctx:        ctx,
		mtx:        sync.Mutex{},
		mgr:        mgr,
		client:     client,
		serializer: serializer,
		watching:   make(map[schema.GroupVersionKind]context.CancelFunc),
		watchCh:    make(chan ResourceInfo, 1024),
		cancelCh:   make(chan ResourceInfo, 1024),
	}
}

func (s *statusManager) WatchStatus(info ResourceInfo) error {
	if !s.started {
		return fmt.Errorf("status manager has not started")
	}

	select {
	case s.watchCh <- info:
	default:
		return fmt.Errorf("the wathCh of status manager is full, drop the info %s", info)
	}

	return nil
}

func (s *statusManager) CancelWatch(info ResourceInfo) error {
	if !s.started {
		return fmt.Errorf("status manager has not started")
	}

	select {
	case s.cancelCh <- info:
	default:
		return fmt.Errorf("the cancelCh of status manager is full, drop the info %s", info)
	}

	return nil
}

func (s *statusManager) Start() {
	s.started = true
	go s.watchStatusWorker()
	go s.cancelWatchWorker()
	go s.waitForTerminatingWorkers()
	go wait.Until(s.watchControllersGC, 5*time.Minute, s.ctx.Done())
}

func (s *statusManager) watchStatusWorker() {
	for info := range s.watchCh {
		if s.isWatching(info) {
			continue
		}
		ctx, cancel := context.WithCancel(s.ctx)
		s.markAsWatching(info, cancel)
		if err := s.startToWatch(ctx, info); err != nil {
			s.unmarkWatching(info)
			klog.Errorf("failed to start to watch status for gvk %s, %v", info.Name, err)
			// TODO:
			// if need retry
			continue
		}
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

func (s *statusManager) isWatching(info ResourceInfo) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	gvk := infoToGVK(info)
	_, ok := s.watching[gvk]
	return ok
}

func (s *statusManager) markAsWatching(info ResourceInfo, cancel context.CancelFunc) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	gvk := infoToGVK(info)
	s.watching[gvk] = cancel
}

func (s *statusManager) unmarkWatching(info ResourceInfo) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if cancel := s.watching[infoToGVK(info)]; cancel != nil {
		cancel()
	}
	delete(s.watching, infoToGVK(info))
}

func (s *statusManager) startToWatch(ctx context.Context, info ResourceInfo) error {
	gvk := infoToGVK(info)
	controllerName := fmt.Sprintf("status-controller-for-%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
	controller, err := controller.NewUnmanaged(controllerName, s.mgr, controller.Options{
		Reconciler: &statusReconciler{Client: s.client, GroupVersionKind: gvk, Serializer: s.serializer},
	})
	if err != nil {
		klog.Errorf("failed to get new unmanaged controller for gvk %s, %v", gvk, err)
		return err
	}

	watchObj := &unstructured.Unstructured{}
	watchObj.SetGroupVersionKind(gvk)
	if err := controller.Watch(&source.Kind{Type: watchObj}, &handler.EnqueueRequestForOwner{
		OwnerType:    &groupingv1alpha1.EdgeApplication{},
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
	edgeAppList := &groupingv1alpha1.EdgeApplicationList{}
	if err := s.client.List(s.ctx, edgeAppList); err != nil {
		klog.Errorf("failed to list EdgeApplication")
		return
	}

	infoMap := make(map[ResourceInfo]struct{})
	for _, edgeApp := range edgeAppList.Items {
		infos, err := GetContainedResourceInfos(&edgeApp, s.serializer)
		if err != nil {
			klog.Errorf("failed to get resourceInfos from edgeApp %s/%s, %v", edgeApp.Namespace, edgeApp.Name, err)
			continue
		}

		for _, info := range infos {
			infoMap[info] = struct{}{}
		}
	}

	for info := range infoMap {
		if err := s.CancelWatch(info); err != nil {
			klog.Errorf("failed to cancel watch for gvk %s, %v", infoToGVK(info), err)
			continue
		}
		klog.V(4).Infof("cancel watching status for gvk %s", infoToGVK(info))
	}
}

func infoToGVK(info ResourceInfo) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   info.Group,
		Version: info.Version,
		Kind:    info.Kind,
	}
}

func GetContainedResourceInfos(edgeApp *groupingv1alpha1.EdgeApplication, yamlSerializer runtime.Serializer) ([]ResourceInfo, error) {
	objs, err := GetContainedResourceObjs(edgeApp, yamlSerializer)
	if err != nil {
		return nil, fmt.Errorf("failed to get contained objs, %v", err)
	}
	infos := []ResourceInfo{}
	for _, obj := range objs {
		gvk := obj.GroupVersionKind()
		info := ResourceInfo{
			Group:     gvk.Group,
			Version:   gvk.Version,
			Kind:      gvk.Kind,
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func GetContainedResourceObjs(edgeApp *groupingv1alpha1.EdgeApplication, yamlSerializer runtime.Serializer) ([]*unstructured.Unstructured, error) {
	objs := []*unstructured.Unstructured{}
	for _, manifest := range edgeApp.Spec.WorkloadTemplate.Manifests {
		obj := &unstructured.Unstructured{}
		_, _, err := yamlSerializer.Decode(manifest.Raw, nil, obj)
		if err != nil {
			return nil, fmt.Errorf("failed to decode manifest of edgeapp %s/%s, %v, manifest: %s",
				edgeApp.Namespace, edgeApp.Name, err, manifest)
		}
		objs = append(objs, obj)
	}
	return objs, nil
}
